package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type discordPayload struct {
	Content string `json:"content"`
}

func SendDiscordAlert(webhookURL, message string) error {
	payload := discordPayload{Content: message}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error serializando payload de discord: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creando request de discord: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error enviando alerta a discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord respondió con status: %d", resp.StatusCode)
	}

	return nil
}
