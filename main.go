package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"relevamiento/core"
	"relevamiento/repository"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/manifoldco/promptui"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error: archivo .env no encontrado")
	}


	db, err := initDB()
	if err != nil {
		log.Fatalf("Error de conexión: %v", err)
	}
	defer db.Close()


	computerName := getEnv("COMPUTERNAME", "Desconocido")

	macAddress := core.GetMacAddress()
	ipAddress := getIPAddress()

	
	if macAddress == "No disponible" || ipAddress == "No disponible" {
		log.Fatal("Error: No se pudo obtener MAC o IP del equipo")
	}

	domainInfo := core.GetDomainInfo()


	if domainInfo.EsMecLocal {
    fmt.Println("En dominio mec.local")
	} else if domainInfo.EnDominio {
    fmt.Printf("En dominio %s\n", domainInfo.NombreDominio)
	} else {
    fmt.Println("WORKGROUP")
	}

piso := "1"
oficina := "Biblioteca"



	fmt.Printf("\nEquipo: %s\n", computerName)


	equipoInfo := repository.EquipoInfo{
		FechaRelevamiento: time.Now().Format("2006-01-02 15:04:05"),
		ComputerName:      computerName,
		NombreAnterior:    computerName,
		MacAddress:        macAddress,
		IPAddress:         ipAddress,
		Piso:              piso,
		Oficina:           oficina,
	}


	fmt.Println(" Guardando...")
	result, err := repository.CreateEquiposRepository(db, equipoInfo)
	if err != nil || !result.Success {
		log.Fatalf(" Error al guardar: %s", result.ErrorMessage)
	}


	fmt.Println("REGISTRO EXITOSO")
	if result.VerifiedData != nil {
		v := result.VerifiedData
		fmt.Printf("ID: %d | %s | %s\n", v.ID, v.ComputerName, v.Oficina)
	}

	fmt.Println("\nPresione Enter para salir...")
	fmt.Scanln()
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
	return result
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
