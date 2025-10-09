package core

import (
	"os/exec"
	"strings"
)

func isValidMacFormat(mac string) bool {
	mac = strings.TrimSpace(mac)

	if len(mac) == 17 && strings.Count(mac, "-") == 5 {
		return true
	}

	if len(mac) == 17 && strings.Count(mac, ":") == 5 {
		return true
	}

	return false
}

func GetMacAddress() string {
	cmd := exec.Command("getmac", "/v", "/fo", "csv")
	out, err := cmd.Output()
	if err != nil {
		return "No disponible"
	}

	lines := strings.Split(string(out), "\n")

	for i, line := range lines {

		if i == 0 || line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 4 {
			continue
		}

		for j := range fields {
			fields[j] = strings.Trim(fields[j], "\"")
			fields[j] = strings.TrimSpace(fields[j])
		}

		nombre := strings.ToLower(fields[0])
		adaptador := strings.ToLower(fields[1])
		macAddr := fields[2]
		transporte := strings.ToLower(fields[3])

		if strings.Contains(transporte, "medios desconectados") ||
			strings.Contains(transporte, "media disconnected") {
			continue
		}

		isEthernet := (strings.Contains(nombre, "ethernet") ||
			strings.Contains(adaptador, "ethernet") ||
			strings.Contains(adaptador, "realtek") ||
			strings.Contains(adaptador, "intel") ||
			strings.Contains(adaptador, "broadcom") ||
			strings.Contains(adaptador, "marvell")) &&
			!strings.Contains(nombre, "wi-fi") &&
			!strings.Contains(adaptador, "wi-fi")

		isExcluded := strings.Contains(nombre, "vmware") ||
			strings.Contains(nombre, "virtualbox") ||
			strings.Contains(nombre, "hyper-v") ||
			strings.Contains(nombre, "vpn") ||
			strings.Contains(adaptador, "virtual") ||
			strings.Contains(adaptador, "fortinet")

		if isEthernet && !isExcluded && isValidMacFormat(macAddr) {
			return macAddr
		}
	}

	return "No disponible"
}
