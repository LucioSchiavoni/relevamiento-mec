package core

import (
	"net"
	"os/exec"
	"strings"
)

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