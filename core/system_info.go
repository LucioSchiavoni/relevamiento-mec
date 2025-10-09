package core

import (
	"os/exec"
	"runtime"
	"strings"
)

type SystemInfo struct {
	OS           string
	Version      string
	Architecture string
	MemoryRAM    string
	Processor    string
	CurrentUser  string
	Manufacturer string
	Model        string
	SerialNumber string
	BIOSVersion  string
}

func GetSystemInfo() SystemInfo {
	info := SystemInfo{
		Architecture: runtime.GOARCH,
	}

	cmd := exec.Command("systeminfo")
	out, err := cmd.Output()
	if err != nil {
		return info
	}

	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Nombre del sistema operativo:") ||
			strings.HasPrefix(line, "OS Name:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.OS = strings.TrimSpace(parts[1])
			}
		}

		if strings.HasPrefix(line, "Version del sistema operativo:") ||
			strings.HasPrefix(line, "OS Version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Version = strings.TrimSpace(parts[1])
			}
		}

		if strings.HasPrefix(line, "Memoria fisica total:") ||
			strings.HasPrefix(line, "Total Physical Memory:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.MemoryRAM = strings.TrimSpace(parts[1])
			}
		}

		if strings.HasPrefix(line, "Procesador(es):") ||
			strings.HasPrefix(line, "Processor(s):") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				processorInfo := strings.TrimSpace(parts[1])
				if !strings.Contains(processorInfo, "instalados") &&
					!strings.Contains(processorInfo, "installed") {
					info.Processor = processorInfo
				}
			}
		}

		if strings.HasPrefix(line, "Fabricante del sistema:") ||
			strings.HasPrefix(line, "System Manufacturer:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Manufacturer = strings.TrimSpace(parts[1])
			}
		}

		if strings.HasPrefix(line, "Modelo del sistema:") ||
			strings.HasPrefix(line, "System Model:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Model = strings.TrimSpace(parts[1])
			}
		}
	}

	if info.Processor == "" {
		info.Processor = getProcessorWMIC()
	}

	info.CurrentUser = getRealUser()

	return info
}

func getProcessorWMIC() string {
	cmd := exec.Command("wmic", "cpu", "get", "name")
	out, err := cmd.Output()
	if err != nil {
		return "Desconocido"
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" || strings.ToLower(line) == "name" {
			continue
		}
		return line
	}

	return "Desconocido"
}

func getRealUser() string {
	currentUser := getCurrentUser()

	if currentUser == "Desconocido" {
		return currentUser
	}

	if isAdministrator(currentUser) {
		lastUser := getLastNonAdminUser()
		if lastUser != "" && lastUser != currentUser {
			return lastUser
		}
	}

	return currentUser
}

func getCurrentUser() string {
	cmd := exec.Command("whoami")
	out, err := cmd.Output()
	if err != nil {
		return "Desconocido"
	}

	return strings.TrimSpace(string(out))
}

func isAdministrator(username string) bool {
	username = strings.ToLower(username)

	if strings.Contains(username, "administrador") ||
		strings.Contains(username, "administrator") ||
		strings.Contains(username, "admin") {
		return true
	}

	cmd := exec.Command("net", "localgroup", "administradores")
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("net", "localgroup", "administrators")
		out, err = cmd.Output()
		if err != nil {
			return false
		}
	}

	outputStr := strings.ToLower(string(out))
	userParts := strings.Split(username, "\\")
	userName := username
	if len(userParts) > 1 {
		userName = userParts[1]
	}

	return strings.Contains(outputStr, strings.ToLower(userName))
}

func getLastNonAdminUser() string {
	cmd := exec.Command("wmic", "netlogin", "get", "name")
	out, err := cmd.Output()
	if err != nil {
		return getLastUserFromRegistry()
	}

	lines := strings.Split(string(out), "\n")
	users := make([]string, 0)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" || strings.ToLower(line) == "name" {
			continue
		}

		if line != "" && !isSystemUser(line) {
			users = append(users, line)
		}
	}

	for i := len(users) - 1; i >= 0; i-- {
		user := users[i]
		if !isAdministrator(user) {
			return user
		}
	}

	return ""
}

func getLastUserFromRegistry() string {
	cmd := exec.Command("reg", "query",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Authentication\\LogonUI",
		"/v", "LastLoggedOnUser")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "LastLoggedOnUser") && strings.Contains(line, "REG_SZ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				lastUser := parts[len(parts)-1]
				if !isSystemUser(lastUser) && !isAdministrator(lastUser) {
					return lastUser
				}
			}
		}
	}

	return ""
}

func isSystemUser(username string) bool {
	username = strings.ToLower(username)

	systemUsers := []string{
		"system",
		"local service",
		"network service",
		"dwm-",
		"umfd-",
		"font driver host",
		"window manager",
	}

	for _, sysUser := range systemUsers {
		if strings.Contains(username, sysUser) {
			return true
		}
	}

	return false
}

func GetBIOSInfo() (string, string) {
	serialNumber := ""
	biosVersion := ""

	cmdSerial := exec.Command("wmic", "bios", "get", "serialnumber")
	outSerial, err := cmdSerial.Output()
	if err == nil {
		lines := strings.Split(string(outSerial), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if i == 0 || line == "" || strings.ToLower(line) == "serialnumber" {
				continue
			}
			serialNumber = line
			break
		}
	}

	cmdVersion := exec.Command("wmic", "bios", "get", "version")
	outVersion, err := cmdVersion.Output()
	if err == nil {
		lines := strings.Split(string(outVersion), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if i == 0 || line == "" || strings.ToLower(line) == "version" {
				continue
			}
			biosVersion = line
			break
		}
	}

	return serialNumber, biosVersion
}
