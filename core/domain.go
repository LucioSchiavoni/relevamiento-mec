package core

import (
	"os/exec"
	"strings"
)

type DomainInfo struct {
	EnDominio     bool
	NombreDominio string
	EsMecLocal    bool
}

func GetDomainInfo() DomainInfo {
	info := DomainInfo{
		EnDominio:     false,
		NombreDominio: "",
		EsMecLocal:    false,
	}

	cmd := exec.Command("wmic", "computersystem", "get", "domain")
	out, err := cmd.Output()
	if err != nil {
		return info
	}

	lines := strings.Split(string(out), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if i == 0 || line == "" || strings.ToLower(line) == "domain" {
			continue
		}

		dominio := strings.ToLower(line)

		if dominio == "workgroup" {
			info.EnDominio = false
			info.NombreDominio = ""
			return info
		}

		info.EnDominio = true
		info.NombreDominio = line

		if dominio == "mec.local" {
			info.EsMecLocal = true
		}

		return info
	}

	return info
}

func GetDomainInfoAlternative() DomainInfo {
	info := DomainInfo{
		EnDominio:     false,
		NombreDominio: "",
		EsMecLocal:    false,
	}

	cmd := exec.Command("systeminfo")
	out, err := cmd.Output()
	if err != nil {
		return info
	}

	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Dominio:") || strings.HasPrefix(line, "Domain:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dominio := strings.TrimSpace(parts[1])
				dominioLower := strings.ToLower(dominio)

				if dominioLower == "workgroup" {
					return info
				}

				info.EnDominio = true
				info.NombreDominio = dominio

				if dominioLower == "mec.local" {
					info.EsMecLocal = true
				}

				return info
			}
		}
	}

	return info
}
