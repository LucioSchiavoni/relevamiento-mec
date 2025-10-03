package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
	"github.com/manifoldco/promptui"
	"relevamiento/core"
	"relevamiento/repository"
)

func main() {

    err := godotenv.Load()
	
    if err != nil {
		log.Fatal("Error cargando archivo .env")
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=10s",
		dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error conectando a la DB: %v", err)
	}
	defer db.Close()


	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("❌ No se puede conectar a la base de datos: %v", err)
	}
	fmt.Println("✅ Conexión a la base de datos establecida correctamente")
	fmt.Println()

	computerName := os.Getenv("COMPUTERNAME")
	if computerName == "" {
		computerName = "Desconocido"
	}

	macAddress := core.GetMacAddress()
	ipAddress := getIPAddress()

	fmt.Println("=== INFORMACIÓN DE RED ===")
	fmt.Printf("IP que se registrará: %s\n", ipAddress)
	fmt.Printf("MAC que se registrará: %s\n", macAddress)\

	if ipAddress != "No disponible" {
		networkInfo := getNetworkInfo(ipAddress)
		fmt.Printf("Red detectada: %s\n", networkInfo)
		fmt.Printf("Tipo de red: %s\n", getNetworkType(ipAddress))
	}
	fmt.Println("========================")
	fmt.Println()

	fmt.Printf("Firewall: %s\n", getSimpleFirewallStatus())
	fmt.Printf("Dominio: %s\n", getDomainStatus())

	piso := "3"
	fmt.Printf("Piso (por defecto): %s\n", piso)

	oficina := inputPrompt("Ingrese la oficina")
	oficina = sanitizeInput(oficina, 50)

	cambiarNombre := inputPrompt("Su nombre es " + computerName + " ¿Desea ingresar un nuevo nombre para la PC? (S/N)")
	var nombreNuevo string
	if strings.ToLower(cambiarNombre) == "s" || strings.ToLower(cambiarNombre) == "si" {
		nombreNuevo = inputPrompt("Ingrese el nuevo nombre de la PC")
		nombreNuevo = sanitizeInput(nombreNuevo, 100)

		if !isValidComputerName(nombreNuevo) {
			fmt.Println("⚠️  Advertencia: El nombre contiene caracteres no permitidos. Se usará el nombre actual.")
			nombreNuevo = computerName
		}
	} else {
		nombreNuevo = computerName
	}

	fecha := time.Now().Format("2006-01-02 15:04:05")

	equipoInfo := repository.EquipoInfo{
		FechaRelevamiento: fecha,
		ComputerName:      nombreNuevo,
		NombreAnterior:    computerName,
		MacAddress:        macAddress,
		IPAddress:         ipAddress,
		Piso:              piso,
		Oficina:           oficina,
	}

	fmt.Println()
	fmt.Println("⏳ Guardando información en la base de datos...")

	result, err := repository.CreateEquiposRepository(db, equipoInfo)
	if err != nil {
		log.Fatalf("❌ ERROR CRÍTICO: %s", result.ErrorMessage)
	}

	if result.Success {
		fmt.Println()
		fmt.Println("╔═══════════════════════════════════════════════════════╗")
		fmt.Println("║     ✅ DATOS GUARDADOS Y VERIFICADOS EXITOSAMENTE     ║")
		fmt.Println("╚═══════════════════════════════════════════════════════╝")
		fmt.Println()

		if result.VerifiedData != nil {
			v := result.VerifiedData
			fmt.Printf("📋 Registro ID: %d\n", v.ID)
			fmt.Printf("💻 Nombre PC: %s\n", v.ComputerName)
			fmt.Printf("🏢 Oficina: %s\n", v.Oficina)
			fmt.Printf("📍 Piso: %s\n", v.Piso)
			fmt.Printf("🌐 IP: %s\n", v.IPAddress)
			fmt.Printf("🔧 MAC: %s\n", v.MacAddress)
			fmt.Printf("📅 Fecha: %s\n", fecha)

			if computerName != nombreNuevo {
				fmt.Printf("📝 Nombre anterior: %s\n", computerName)
			}
		}

		fmt.Println()
		fmt.Printf("✅ Total de filas insertadas: %d\n", result.RowsAffected)
		if result.InsertedID > 0 {
			fmt.Printf("🆔 ID del nuevo registro: %d\n", result.InsertedID)
		}
		fmt.Println()
		fmt.Println("✅ El registro fue confirmado en la base de datos.")
	} else {
		log.Fatalf("❌ ERROR: %s", result.ErrorMessage)
	}

	fmt.Println()
	fmt.Println("Presione Enter para salir...")
	fmt.Scanln()
}


func inputPrompt(label string) string {
	prompt := promptui.Prompt{
		Label: label,
	}
	result, err := prompt.Run()
	if err != nil {
		fmt.Println("Error leyendo input:", err)
		os.Exit(1)
	}
	return result
}

func getIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "No disponible"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				if !strings.HasPrefix(ip, "169.254.") {
					return ip
				}
			}
		}
	}
	return "No disponible"
}

func getSimpleFirewallStatus() string {
	cmd := exec.Command("powershell", "-Command", "Get-NetFirewallProfile | Select-Object Name, Enabled | Format-Table -HideTableHeaders")
	output, err := cmd.Output()
	if err != nil {
		return "Error al obtener estado del firewall"
	}
	result := strings.TrimSpace(string(output))
	if strings.Contains(result, "True") {
		return "Activado"
	}
	return "Desactivado"
}

func getDomainStatus() string {
	cmd := exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_ComputerSystem).PartOfDomain")
	output, err := cmd.Output()
	if err != nil {
		return "Error al verificar dominio"
	}

	result := strings.TrimSpace(string(output))
	if result == "True" {
		cmdDomain := exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_ComputerSystem).Domain")
		domainOutput, err := cmdDomain.Output()
		if err != nil {
			return "Sí (dominio desconocido)"
		}
		domain := strings.TrimSpace(string(domainOutput))
		return fmt.Sprintf("Sí - Dominio: %s", domain)
	}
	return "No (Grupo de trabajo)"
}

func getNetworkInfo(ip string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "No se pudo obtener información de red"
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.String() == ip {
					network := ipnet.IP.Mask(ipnet.Mask)
					maskSize, _ := ipnet.Mask.Size()
					return fmt.Sprintf("%s/%d (Interfaz: %s)", network.String(), maskSize, iface.Name)
				}
			}
		}
	}

	return "Red no identificada"
}

func getNetworkType(ip string) string {
	if strings.HasPrefix(ip, "192.168.") {
		return "Red privada (Clase C)"
	} else if strings.HasPrefix(ip, "10.") {
		return "Red privada (Clase A)"
	} else if strings.HasPrefix(ip, "172.") {
		parts := strings.Split(ip, ".")
		if len(parts) >= 2 {
			second := parts[1]
			if second >= "16" && second <= "31" {
				return "Red privada (Clase B)"
			}
		}
		return "Red corporativa/institucional"
	} else if strings.HasPrefix(ip, "169.254.") {
		return "APIPA (Autoconfiguración)"
	} else {
		return "Red pública/externa"
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func sanitizeInput(input string, maxLength int) string {
	input = strings.TrimSpace(input)

	if len(input) > maxLength {
		input = input[:maxLength]
	}

	input = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, input)

	return input
}

func isValidComputerName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_]{1,15}$`, name)
	return matched
}