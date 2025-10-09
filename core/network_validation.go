package core

import (
	"fmt"
	"os/exec"
	"strings"
)

type NetworkAdapter struct {
	Name        string
	AdapterType string
	MacAddress  string
	Status      string
	IsEthernet  bool
	IsActive    bool
	Speed       string
	IPAddress   string
}

func GetAllNetworkAdapters() []NetworkAdapter {
	adapters := []NetworkAdapter{}

	cmd := exec.Command("getmac", "/v", "/fo", "csv")
	out, err := cmd.Output()
	if err != nil {
		return adapters
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

		adapter := NetworkAdapter{
			Name:        fields[0],
			AdapterType: fields[1],
			MacAddress:  fields[2],
			Status:      fields[3],
		}

		nameLower := strings.ToLower(adapter.Name)
		typeLower := strings.ToLower(adapter.AdapterType)

		adapter.IsActive = !strings.Contains(strings.ToLower(adapter.Status), "disconnected") &&
			!strings.Contains(strings.ToLower(adapter.Status), "desconectados")

		adapter.IsEthernet = (strings.Contains(nameLower, "ethernet") ||
			strings.Contains(typeLower, "ethernet") ||
			strings.Contains(typeLower, "realtek") ||
			strings.Contains(typeLower, "intel") ||
			strings.Contains(typeLower, "broadcom")) &&
			!strings.Contains(nameLower, "wi-fi") &&
			!strings.Contains(nameLower, "wireless") &&
			!strings.Contains(nameLower, "vmware") &&
			!strings.Contains(nameLower, "virtualbox") &&
			!strings.Contains(typeLower, "virtual")

		if isValidMacFormat(adapter.MacAddress) {
			adapters = append(adapters, adapter)
		}
	}

	return adapters
}

func GetEthernetMacWithConfirmation() (string, error) {
	adapters := GetAllNetworkAdapters()

	ethernetAdapters := []NetworkAdapter{}
	for _, adapter := range adapters {
		if adapter.IsEthernet {
			ethernetAdapters = append(ethernetAdapters, adapter)
		}
	}

	if len(ethernetAdapters) == 0 {
		return "", fmt.Errorf("no se encontraron adaptadores Ethernet")
	}

	fmt.Println("\nAdaptadores Ethernet detectados:")
	for i, adapter := range ethernetAdapters {
		status := "Desconectado"
		if adapter.IsActive {
			status = "Conectado"
		}

		fmt.Printf("  [%d] %s - %s\n", i+1, adapter.MacAddress, status)
	}

	for _, adapter := range ethernetAdapters {
		if adapter.IsActive {
			fmt.Printf("\nMAC seleccionada: %s\n", adapter.MacAddress)
			return adapter.MacAddress, nil
		}
	}

	fmt.Printf("\n[!] Sin adaptadores activos, usando: %s\n", ethernetAdapters[0].MacAddress)
	return ethernetAdapters[0].MacAddress, nil
}

func ValidateNetworkConfiguration() error {
	adapters := GetAllNetworkAdapters()

	hasActiveEthernet := false
	for _, adapter := range adapters {
		if adapter.IsEthernet && adapter.IsActive {
			hasActiveEthernet = true
			break
		}
	}

	if !hasActiveEthernet {
		return fmt.Errorf("[!] ADVERTENCIA: No hay cable Ethernet conectado")
	}

	return nil
}
