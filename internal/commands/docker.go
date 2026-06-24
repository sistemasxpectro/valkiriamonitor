package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var validContainerName = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// DockerContainer mapea la salida JSON de 'docker ps --format "{{json .}}"'
type DockerContainer struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

func GetDockerStatus() (string, error) {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{json .}}")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Si docker no está corriendo o el socket no tiene permisos
		return "", fmt.Errorf("error ejecutando docker ps (revisa el volumen docker.sock): %w", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return "No hay contenedores de Docker en este host.", nil
	}

	lines := strings.Split(output, "\n")
	var result string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var container DockerContainer
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			continue // Ignorar líneas mal formadas
		}

		statusEmoji := "🟢"
		if container.State != "running" {
			statusEmoji = "🔴"
		}

		result += fmt.Sprintf("%s %s\nImagen: %s\nEstado: %s (%s)\n\n",
			statusEmoji, container.Names, container.Image, container.State, container.Status)
	}

	return result, nil
}

func ManageDockerContainer(action, containerName string) (string, error) {
	// Validar que el nombre sea seguro antes de pasarlo al sistema
	if !validContainerName.MatchString(containerName) {
		return "", fmt.Errorf("nombre de contenedor inválido: '%s'. Solo se permiten letras, números, puntos, guiones y guiones bajos", containerName)
	}

	// Prevención de suicidio: no detener ni reiniciar el propio contenedor
	if containerName == "valkiria-monitor" && (action == "stop" || action == "restart") {
		return "❌ Acción denegada: por seguridad no puedo detener o reiniciar mi propio contenedor (`valkiria-monitor`).", nil
	}

	validActions := map[string]bool{"start": true, "stop": true, "restart": true}
	if !validActions[action] {
		return "", fmt.Errorf("acción inválida: %s", action)
	}

	cmd := exec.Command("docker", action, containerName)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fallo al ejecutar 'docker %s %s': %w", action, containerName, err)
	}

	actionTranslated := map[string]string{
		"start":   "iniciado",
		"stop":    "detenido",
		"restart": "reiniciado",
	}[action]

	return fmt.Sprintf("✅ Contenedor '%s' %s correctamente.", containerName, actionTranslated), nil
}
