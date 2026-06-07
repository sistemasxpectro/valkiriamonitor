package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
)

// validAppName solo permite nombres alfanuméricos, guiones y guiones bajos
var validAppName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Estructuras para parsear la salida de 'pm2 jlist'
type PM2Process struct {
	Name   string `json:"name"`
	PM2Env struct {
		Status string `json:"status"`
	} `json:"pm2_env"`
	Monit struct {
		Memory int64   `json:"memory"`
		CPU    float64 `json:"cpu"`
	} `json:"monit"`
}

func GetPM2Status() (string, error) {
	cmd := exec.Command("pm2", "jlist")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error ejecutando pm2: %w", err)
	}

	var processes []PM2Process
	if err := json.Unmarshal(out.Bytes(), &processes); err != nil {
		return "", fmt.Errorf("error parseando JSON de pm2: %w", err)
	}

	if len(processes) == 0 {
		return "No hay procesos en PM2\\.", nil
	}

	// Formateamos la salida para Telegram (MarkdownV2)
	var result string
	for _, p := range processes {
		statusEmoji := "🟢"
		if p.PM2Env.Status != "online" {
			statusEmoji = "🔴"
		}
		
		memMB := float64(p.Monit.Memory) / 1024 / 1024
		
		// OJO: Los caracteres especiales se escaparán luego en la función general de Telegram
		result += fmt.Sprintf("%s *%s*\nEstado: %s\nCPU: %.1f%%\nRAM: %.1f MB\n\n",
			statusEmoji, p.Name, p.PM2Env.Status, p.Monit.CPU, memMB)
	}

	return result, nil
}

func RestartPM2App(appName string) (string, error) {
	// Validar que el nombre sea seguro antes de pasarlo al sistema
	if appName != "all" && !validAppName.MatchString(appName) {
		return "", fmt.Errorf("nombre de app inválido: '%s'. Solo se permiten letras, números, guiones y guiones bajos", appName)
	}

	cmd := exec.Command("pm2", "restart", appName)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fallo al reiniciar '%s': %w", appName, err)
	}
	return fmt.Sprintf("✅ Proceso '%s' reiniciado correctamente\\.", appName), nil
}
