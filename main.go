package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"relevamiento/core"
	"relevamiento/repository"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/manifoldco/promptui"
)

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("       RELEVAMIENTO DE EQUIPOS")
	fmt.Println(strings.Repeat("=", 60))

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error: archivo .env no encontrado")
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("Error de conexion a DB: %v", err)
	}
	defer db.Close()

	computerName := getEnv("COMPUTERNAME", "Desconocido")

	macAddress, err := core.GetEthernetMacWithConfirmation()
	if err != nil || macAddress == "No disponible" {
		log.Fatal("[ERROR] No se pudo obtener MAC de Ethernet")
	}

	ipAddress := getIPAddress()
	if ipAddress == "No disponible" {
		log.Fatal("[ERROR] No se pudo obtener IP del equipo")
	}

	domainInfo := core.GetDomainInfo()
	if !domainInfo.EnDominio {
		fmt.Println("\n[!] ADVERTENCIA: EQUIPO NO ESTA EN DOMINIO")
	}

	piso, oficina := getLocationData()

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
		log.Fatalf("[ERROR] %s", result.ErrorMessage)
	}

	printSuccess(result)

	fmt.Println("\nPresione Enter para salir...")
	fmt.Scanln()
}

func getLocationData() (string, string) {
	config, _ := core.LoadLocationConfig()

	fmt.Println("\n" + strings.Repeat("=", 60))

	if config != nil {
		fmt.Printf("Configuracion actual: Piso %s - %s\n", config.Piso, config.Oficina)
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
			return config.Piso, config.Oficina
		} else if opcion == "2" {
			return configureLocation()
		} else {
			fmt.Println("[X] Opcion invalida")
		}
	}
}

func configureLocation() (string, string) {
	piso := inputPrompt("\nPISO")
	if piso == "" {
		piso = "0"
	}

	oficina := inputPrompt("OFICINA")
	if oficina == "" {
		log.Fatal("[ERROR] Oficina es obligatoria")
	}

	if err := core.SaveLocationConfig(piso, oficina); err != nil {
		fmt.Printf("[!] No se pudo guardar: %v\n", err)
	} else {
		fmt.Printf("[OK] Guardado: Piso %s - %s\n", piso, oficina)
	}

	return piso, oficina
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=10s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute * 3)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func getIPAddress() string {
	addrs, err := net.InterfaceAddrs()

	networkPrefix := os.Getenv("NETWORK_PREFIX")
	if networkPrefix == "" {
		log.Fatal("Error: NETWORK_PREFIX no configurado en .env")
	}

	if err != nil {
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

func inputPrompt(label string) string {
	prompt := promptui.Prompt{Label: label}
	result, err := prompt.Run()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(result)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
