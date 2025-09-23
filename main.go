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

    targetHost := "mec.local"
	lookupIPs, err := net.LookupHost(targetHost)
	if err != nil || len(lookupIPs) == 0 {
		fmt.Printf("❌ No se pudo resolver '%s' por DNS o no devolvió IPs: %v\n", targetHost, err)
		fmt.Println("No se puede continuar sin la IP asociada a", targetHost)
		os.Exit(1)
	}
	
	for i := range lookupIPs {
		lookupIPs[i] = strings.TrimSpace(lookupIPs[i])
	}

	matchIP, matchMAC := findLocalIPMatching(lookupIPs)
	if matchIP == "" {
		fmt.Println("❌ Ninguna IP local coincide con las IPs de", targetHost)
		fmt.Println("IPs resueltas: ", strings.Join(lookupIPs, ", "))
		fmt.Println("Listando IPs locales para ayuda:")
		printLocalIPs()
		fmt.Println("No se puede continuar sin la IP de", targetHost)
		os.Exit(1)
	}

    func printLocalIPs() {
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("  Error listando interfaces:", err)
		return
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		if len(addrs) == 0 {
			continue
		}
		fmt.Printf(" - %s (%s):\n", iface.Name, iface.HardwareAddr.String())
		for _, a := range addrs {
			fmt.Printf("    %s\n", a.String())
		}
	}
}

     
    func findLocalIPMatching(lookupIPs []string) (string, string) {

	lookupSet := map[string]bool{}
	for _, ip := range lookupIPs {
		lookupSet[ip] = true
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", ""
	}

	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue 
			}
			ipStr := ip.String()
			if lookupSet[ipStr] {
				mac := iface.HardwareAddr.String()
				if mac == "" {
					mac = "No disponible"
				}
				return ipStr, mac
			}
		}
	}
	return "", ""
}

    fmt.Printf("Firewall: %s\n", getSimpleFirewallStatus())
    fmt.Printf("Dominio: %s\n", getDomainStatus())

    var piso string 
    for {
        fmt.Print("Ingrese el piso: ")
        fmt.Scanln(&piso)
        if strings.TrimSpace(piso) != "" {
        break 
        }
        fmt.Println("El piso no puede estar vacío.")
    }

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
                ip := ipnet.IP.String()
                // Excluir direcciones APIPA (169.254.x.x)
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