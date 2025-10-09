package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"relevamiento/core"
	"relevamiento/repository"
	"runtime/debug"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var logFile *os.File

func main() {
	defer handlePanic()
	defer waitForExit()

	initLogging()
	defer closeLogging()

	logInfo("Iniciando relevamiento...")

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("       RELEVAMIENTO DE EQUIPOS")
	fmt.Println(strings.Repeat("=", 60))

	if err := validateEnvironment(); err != nil {
		logError("Error en validacion inicial", err)
		log.Fatalf("[ERROR] %v", err)
	}

	if err := godotenv.Load(); err != nil {
		logError("Archivo .env no encontrado", err)
		log.Fatal("Error: archivo .env no encontrado")
	}

	showMenu()
}

func showMenu() {
	config, _ := core.LoadLocationConfig()

	fmt.Println("\n" + strings.Repeat("=", 60))

	if config != nil {
		fmt.Printf("Configuracion actual: Piso %s - %s\n", config.Piso, config.Oficina)
	} else {
		fmt.Println("Sin configuracion guardada")
	}

	fmt.Println("\n[1] Captura rapida")
	fmt.Println("[2] Configurar ubicacion")

	var opcion string
	for {
		fmt.Print("\nOpcion: ")
		fmt.Scanln(&opcion)
		
		if opcion == "1" {
			if config == nil {
				fmt.Println("[X] Debe configurar primero (opcion 2)")
				continue
			}
			executeCapture(config.Piso, config.Oficina)
			return
		} else if opcion == "2" {
			configureOnly()
			return
		} else {
			fmt.Println("[X] Opcion invalida")
		}
	}
}

func configureOnly() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("       CONFIGURAR UBICACION")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Print("\nPISO (presione Enter para usar '0'): ")
	piso, _ := reader.ReadString('\n')
	piso = strings.TrimSpace(piso)
	if piso == "" {
		piso = "0"
	}

	fmt.Print("OFICINA: ")
	oficina, _ := reader.ReadString('\n')
	oficina = strings.TrimSpace(oficina)
	
	if oficina == "" {
		logError("Oficina vacia", nil)
		log.Fatal("[ERROR] La oficina es obligatoria")
	}

	if err := core.SaveLocationConfig(piso, oficina); err != nil {
		logError("No se pudo guardar configuracion", err)
		fmt.Printf("[!] No se pudo guardar: %v\n", err)
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("[OK] CONFIGURACION GUARDADA")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\nPiso:    %s\n", piso)
	fmt.Printf("Oficina: %s\n", oficina)
	fmt.Println(strings.Repeat("=", 60))
	
	logInfo(fmt.Sprintf("Configuracion guardada: Piso %s - %s", piso, oficina))
}

func executeCapture(piso, oficina string) {
	db, err := initDB()
	if err != nil {
		logError("Error de conexion a DB", err)
		log.Fatalf("Error de conexion a DB: %v", err)
	}
	defer db.Close()

	computerName := getEnv("COMPUTERNAME", "Desconocido")
	logInfo(fmt.Sprintf("Computer Name: %s", computerName))
	
	macAddress, err := core.GetEthernetMacWithConfirmation()
	if err != nil || macAddress == "No disponible" {
		logError("No se pudo obtener MAC", err)
		log.Fatal("[ERROR] No se pudo obtener MAC de Ethernet")
	}
	logInfo(fmt.Sprintf("MAC detectada: %s", macAddress))

	ipAddress := getIPAddress()
	if ipAddress == "No disponible" {
		logError("No se pudo obtener IP", nil)
		log.Fatal("[ERROR] No se pudo obtener IP del equipo")
	}
	logInfo(fmt.Sprintf("IP detectada: %s", ipAddress))

	domainInfo := core.GetDomainInfo()
	if !domainInfo.EnDominio {
		fmt.Println("\n[!] ADVERTENCIA: EQUIPO NO ESTA EN DOMINIO")
		logWarning("Equipo no esta en dominio")
	} else {
		logInfo(fmt.Sprintf("Dominio: %s", domainInfo.NombreDominio))
	}

	equipoInfo := repository.EquipoInfo{
		FechaRelevamiento: time.Now().Format("2006-01-02 15:04:05"),
		ComputerName:      computerName,
		NombreAnterior:    computerName,
		MacAddress:        macAddress,
		IPAddress:         ipAddress,
		Piso:              piso,
		Oficina:           oficina,
	}

	fmt.Println("\n>> Guardando...")
	result, err := repository.CreateEquiposRepository(db, equipoInfo)
	if err != nil || !result.Success {
		logError("Error al guardar en DB", err)
		log.Fatalf("[ERROR] %s", result.ErrorMessage)
	}

	logInfo(fmt.Sprintf("Registro exitoso - ID: %d", result.InsertedID))
	printSuccess(result)
}

func handlePanic() {
	if r := recover(); r != nil {
		logError("PANIC DETECTADO", fmt.Errorf("%v", r))
		fmt.Printf("\n[ERROR CRITICO] El programa encontro un error inesperado\n")
		fmt.Printf("Error: %v\n", r)
		fmt.Printf("\nStack trace:\n%s\n", debug.Stack())
		fmt.Printf("\nRevise el archivo error.log para mas detalles\n")
		waitForExit()
	}
}

func waitForExit() {
	fmt.Println("\nPresione Enter para salir...")
	fmt.Scanln()
}

func initLogging() {
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("No se pudo obtener ruta del ejecutable: %v", err)
		return
	}
	
	exeDir := filepath.Dir(exePath)
	logPath := filepath.Join(exeDir, "error.log")

	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("No se pudo crear archivo de log: %v", err)
		return
	}

	log.SetOutput(logFile)
	logInfo("========== NUEVA EJECUCION ==========")
}

func closeLogging() {
	if logFile != nil {
		logFile.Close()
	}
}

func logInfo(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[INFO] %s - %s", timestamp, msg)
	if logFile != nil {
		logFile.WriteString(logMsg + "\n")
	}
	log.Println(logMsg)
}

func logWarning(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[WARNING] %s - %s", timestamp, msg)
	if logFile != nil {
		logFile.WriteString(logMsg + "\n")
	}
	log.Println(logMsg)
}

func logError(msg string, err error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[ERROR] %s - %s", timestamp, msg)
	if err != nil {
		logMsg += fmt.Sprintf(" - %v", err)
	}
	if logFile != nil {
		logFile.WriteString(logMsg + "\n")
	}
	log.Println(logMsg)
}

func validateEnvironment() error {
	computerName := os.Getenv("COMPUTERNAME")
	if computerName == "" {
		return fmt.Errorf("variable COMPUTERNAME no disponible")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo obtener ruta del ejecutable: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	envPath := filepath.Join(exeDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("archivo .env no encontrado en: %s", exeDir)
	}

	testFile := filepath.Join(exeDir, "test_write.tmp")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("no hay permisos de escritura en: %s", exeDir)
	}
	os.Remove(testFile)

	return nil
}

func printSuccess(result *repository.EquipoResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("[OK] REGISTRO EXITOSO")
	fmt.Println(strings.Repeat("=", 60))
	
	if result.VerifiedData != nil {
		v := result.VerifiedData
		fmt.Printf("\nID:        %d\n", v.ID)
		fmt.Printf("Equipo:    %s\n", v.ComputerName)
		fmt.Printf("MAC:       %s\n", v.MacAddress)
		fmt.Printf("IP:        %s\n", v.IPAddress)
		fmt.Printf("Ubicacion: Piso %s - %s\n", v.Piso, v.Oficina)
	}
	
	fmt.Println(strings.Repeat("=", 60))
}

func initDB() (*sql.DB, error) {
	requiredVars := []string{"DB_USER", "DB_PASS", "DB_HOST", "DB_PORT", "DB_NAME"}
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			return nil, fmt.Errorf("variable %s no configurada en .env", varName)
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=10s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error abriendo conexion: %v", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute * 3)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("no se pudo conectar a la base de datos: %v", err)
	}

	logInfo("Conexion a DB exitosa")
	return db, nil
}

func getIPAddress() string {
	addrs, err := net.InterfaceAddrs()

	networkPrefix := os.Getenv("NETWORK_PREFIX")
	if networkPrefix == "" {
		logError("NETWORK_PREFIX no configurado", nil)
		log.Fatal("Error: NETWORK_PREFIX no configurado en .env")
	}

	if err != nil {
		logError("Error obteniendo direcciones de red", err)
		return "No disponible"
	}

	var fallbackIP string

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()

				if strings.HasPrefix(ip, "169.254.") {
					continue
				}

				if strings.HasPrefix(ip, networkPrefix) {
					return ip
				}

				if fallbackIP == "" {
					fallbackIP = ip
				}
			}
		}
	}

	if fallbackIP != "" {
		return fallbackIP
	}

	return "No disponible"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}