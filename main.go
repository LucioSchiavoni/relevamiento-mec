package main

import (
    "database/sql"
    "fmt"
    "log"
    "net"
    "os"
    "os/exec"
    "strings"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/manifoldco/promptui"
)

func main() {
	
    dsn := "root:user@tcp(localhost:3308)/relevamiento_db"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatalf("Error conectando a la DB: %v", err)
    }
    defer db.Close()

    computerName := os.Getenv("COMPUTERNAME")
    if computerName == "" {
        computerName = "Desconocido"
    }

    macAddress := getMacAddress()
    ipAddress := getIPAddress()


    piso := inputPrompt("Ingrese el piso")
    oficina := inputPrompt("Ingrese la oficina")

    // 3️⃣ Preguntar si hay nuevo nombre de PC
    cambiarNombre := inputPrompt("Su nombre es "+computerName+" ¿Desea ingresar un nuevo nombre para la PC? (S/N)")
    var nombreNuevo string
    if strings.ToLower(cambiarNombre) == "s" || strings.ToLower(cambiarNombre) == "si" {
        nombreNuevo = inputPrompt("Ingrese el nuevo nombre de la PC")
    } else {
        nombreNuevo = computerName
    }


    fecha := time.Now().Format("2006-01-02 15:04:05")

    query := `INSERT INTO equipo_info 
        (fecha_relevamiento, computer_name, nombre_anterior, mac_address, ip_address, piso, oficina)
        VALUES (?, ?, ?, ?, ?, ?, ?)`

    _, err = db.Exec(query, fecha, nombreNuevo, computerName, macAddress, ipAddress, piso, oficina)
    if err != nil {
        log.Fatalf("Error insertando en la DB: %v", err)
    }

    fmt.Println("✅ Datos guardados correctamente en la base de datos.")
    fmt.Println("Nombre anterior:", computerName)
    fmt.Println("Nombre guardado:", nombreNuevo)
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

func getMacAddress() string {
    cmd := exec.Command("getmac")
    out, err := cmd.Output()
    if err != nil {
        return "No disponible"
    }
    
    lines := strings.Split(string(out), "\n")

    for _, line := range lines {
        line = strings.TrimSpace(line)
        
        if strings.Contains(line, "Dirección física") || 
           strings.Contains(line, "==========") || 
           line == "" {
            continue
        }
        
        fields := strings.Fields(line)
        if len(fields) >= 2 {
            macAddr := fields[0]
            transportName := strings.Join(fields[1:], " ")
            
            if strings.Contains(macAddr, "-") && 
               len(macAddr) == 17 && 
               !strings.Contains(transportName, "Medios desconectados") {
                return macAddr
            }
        }
    }
    
    return "No disponible"
}

func getIPAddress() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return "No disponible"
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return "No disponible"
}
