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
    "relevamiento/core"
)

func main() {

    
    dsn := "root:user@tcp(172.24.25.4:3308)/relevamiento_db"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatalf("Error conectando a la DB: %v", err)
    }
    defer db.Close()

    computerName := os.Getenv("COMPUTERNAME")
    if computerName == "" {
        computerName = "Desconocido"
    }

    macAddress := core.GetMacAddress()
    ipAddress := getIPAddress()
    
    // Mostrar información de la red que se va a registrar
    fmt.Println("=== INFORMACIÓN DE RED ===")
    fmt.Printf("IP que se registrará: %s\n", ipAddress)
    fmt.Printf("MAC que se registrará: %s\n", macAddress)
    
    if ipAddress != "No disponible" {
        networkInfo := getNetworkInfo(ipAddress)
        fmt.Printf("Red detectada: %s\n", networkInfo)
        fmt.Printf("Tipo de red: %s\n", getNetworkType(ipAddress))
    }
    fmt.Println("========================")
    fmt.Println()

    // targetHost := "mec.local"
	// lookupIPs, err := net.LookupHost(targetHost)
	// if err != nil || len(lookupIPs) == 0 {
	// 	fmt.Printf("❌ No se pudo resolver '%s' por DNS o no devolvió IPs: %v\n", targetHost, err)
	// 	fmt.Println("No se puede continuar sin la IP asociada a", targetHost)
	// 	os.Exit(1)
	// }
	
	// for i := range lookupIPs {
	// 	lookupIPs[i] = strings.TrimSpace(lookupIPs[i])
	// }

	// matchIP, _ := findLocalIPMatching(lookupIPs)
	// if matchIP == "" {
	// 	fmt.Println("❌ Ninguna IP local coincide con las IPs de", targetHost)
	// 	fmt.Println("IPs resueltas: ", strings.Join(lookupIPs, ", "))
	// 	fmt.Println("Listando IPs locales para ayuda:")
	// 	printLocalIPs()
	// 	fmt.Println("No se puede continuar sin la IP de", targetHost)
	// 	os.Exit(1)
	// }

  

    fmt.Printf("Firewall: %s\n", getSimpleFirewallStatus())
    fmt.Printf("Dominio: %s\n", getDomainStatus())

    piso := "1"
    fmt.Printf("Piso detectado (por defecto): %s\n", piso)

    oficina := inputPrompt("Ingrese la oficina")

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
    // Obtener la interfaz de red asociada con la IP
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
                    // Calcular la red basada en la máscara
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
    // Determinar el tipo de red basado en el rango de IP
    if strings.HasPrefix(ip, "192.168.") {
        return "Red privada (Clase C)"
    } else if strings.HasPrefix(ip, "10.") {
        return "Red privada (Clase A)"
    } else if strings.HasPrefix(ip, "172.") {
        // Verificar si está en el rango 172.16.0.0 - 172.31.255.255
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