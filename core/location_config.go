package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LocationConfig struct {
	Piso    string `json:"piso"`
	Oficina string `json:"oficina"`
}

const configFileName = "location_config.json"

func GetConfigFilePath() string {
	exePath, err := os.Executable()
	if err != nil {
		return configFileName
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, configFileName)
}

func LoadLocationConfig() (*LocationConfig, error) {
	configPath := GetConfigFilePath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error leyendo configuracion: %v", err)
	}

	var config LocationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parseando configuracion: %v", err)
	}

	return &config, nil
}

func SaveLocationConfig(piso, oficina string) error {
	config := LocationConfig{
		Piso:    piso,
		Oficina: oficina,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando configuracion: %v", err)
	}

	configPath := GetConfigFilePath()
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error guardando configuracion: %v", err)
	}

	return nil
}

func DeleteLocationConfig() error {
	configPath := GetConfigFilePath()
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error eliminando configuracion: %v", err)
	}
	return nil
}

func HasLocationConfig() bool {
	configPath := GetConfigFilePath()
	_, err := os.Stat(configPath)
	return err == nil
}
