package notifier

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"valkiria-monitor/internal/metrics"
)

type TelegramService struct {
	bot         *tgbotapi.BotAPI
	adminChatID int64
}

func NewTelegramService(token string, adminChatID int64) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("error iniciando bot de telegram: %w", err)
	}
	return &TelegramService{
		bot:         bot,
		adminChatID: adminChatID,
	}, nil
}

// EscapeMD2 escapa los caracteres especiales de MarkdownV2
func (s *TelegramService) EscapeMD2(text string) string {
	chars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range chars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func (s *TelegramService) SendCriticalAlert(message string) error {
	escapedMsg := s.EscapeMD2(message)
	// Para las alertas proactivas podemos usar MarkdownV2, pero como hemos escapado todo, 
	// agregamos emojis e indicativos sin markup interno o armamos el string con formato manualmente.
	// Aquí usamos el mensaje ya escapado y añadimos negrita en un prefijo.
	finalText := "⚠️ *ALERTA CRÍTICA* ⚠️\n\n" + escapedMsg

	msg := tgbotapi.NewMessage(s.adminChatID, finalText)
	msg.ParseMode = "MarkdownV2"
	
	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error enviando alerta crítica por telegram: %w", err)
	}
	return nil
}

func (s *TelegramService) StartListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		switch update.Message.Command() {
		case "statusvps":
			stats, err := metrics.GetStats()
			if err != nil {
				log.Printf("Error obteniendo métricas: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error obteniendo métricas del sistema.")
				s.bot.Send(msg)
				continue
			}

			text := fmt.Sprintf(
				"🖥 *Estado del VPS*\n\n"+
					"⚙️ *CPU:* %s\n"+
					"🧠 *RAM:* %s\n"+
					"💽 *Disco:* %s\n"+
					"⏱ *Uptime:* %s",
				s.EscapeMD2(fmt.Sprintf("%.2f%%", stats.CPUUsage)),
				s.EscapeMD2(fmt.Sprintf("%.2f%% (%d MB / %d MB)", stats.RAMUsage, stats.RAMUsed/(1024*1024), stats.RAMTotal/(1024*1024))),
				s.EscapeMD2(fmt.Sprintf("%.2f%% (%d GB / %d GB)", stats.DiskUsage, stats.DiskUsed/(1024*1024*1024), stats.DiskTotal/(1024*1024*1024))),
				s.EscapeMD2(fmt.Sprintf("%.2f horas", stats.UptimeHours)),
			)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = "MarkdownV2"
			if _, err := s.bot.Send(msg); err != nil {
				log.Printf("Error enviando respuesta de statusvps: %v", err)
			}
		}
	}
}
