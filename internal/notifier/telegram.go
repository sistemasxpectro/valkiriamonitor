package notifier

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"valkiria-monitor/internal/commands"
	"valkiria-monitor/internal/metrics"
)

type TelegramService struct {
	bot         *tgbotapi.BotAPI
	adminChatID int64
	serverName  string
}

func NewTelegramService(token string, adminChatID int64, serverName string) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("error iniciando bot de telegram: %w", err)
	}
	return &TelegramService{
		bot:         bot,
		adminChatID: adminChatID,
		serverName:  serverName,
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

func (s *TelegramService) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	if _, err := s.bot.Send(msg); err != nil {
		log.Printf("Error enviando respuesta: %v", err)
	}
}

func (s *TelegramService) StartListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		// [SEGURIDAD] Ignorar cualquier mensaje que no venga del Administrador
		if update.Message.Chat.ID != s.adminChatID {
			log.Printf("[Seguridad] Intento de acceso denegado del ChatID: %d", update.Message.Chat.ID)
			continue
		}

		command := update.Message.Command()
		chatID := update.Message.Chat.ID

		switch command {
		case "statusvps":
			stats, err := metrics.GetStats()
			if err != nil {
				log.Printf("Error obteniendo métricas: %v", err)
				s.reply(chatID, s.EscapeMD2("Error obteniendo métricas del sistema."))
				continue
			}

			text := fmt.Sprintf(
				"🖥 *Estado del Servidor: %s*\n\n"+
					"⚙️ *CPU:* %s\n"+
					"🧠 *RAM:* %s\n"+
					"💽 *Disco:* %s\n"+
					"⏱ *Uptime:* %s",
				s.EscapeMD2(s.serverName),
				s.EscapeMD2(fmt.Sprintf("%.2f%%", stats.CPUUsage)),
				s.EscapeMD2(fmt.Sprintf("%.2f%% (%d MB / %d MB)", stats.RAMUsage, stats.RAMUsed/(1024*1024), stats.RAMTotal/(1024*1024))),
				s.EscapeMD2(fmt.Sprintf("%.2f%% (%d GB / %d GB)", stats.DiskUsage, stats.DiskUsed/(1024*1024*1024), stats.DiskTotal/(1024*1024*1024))),
				s.EscapeMD2(fmt.Sprintf("%.2f horas", stats.UptimeHours)),
			)
			s.reply(chatID, text)

		case "pm2status":
			statusMsg, err := commands.GetPM2Status()
			if err != nil {
				log.Printf("[PM2] Error: %v", err)
				s.reply(chatID, s.EscapeMD2("Error consultando PM2. Revisa los logs."))
				continue
			}
			s.reply(chatID, s.EscapeMD2("📊 *Estado de PM2:*\n\n")+s.EscapeMD2(statusMsg))

		case "pm2restart":
			args := update.Message.CommandArguments()
			if args == "" {
				s.reply(chatID, s.EscapeMD2("⚠️ Debes especificar el nombre de la app o 'all'. Ejemplo: /pm2restart api"))
				continue
			}
			
			resultMsg, err := commands.RestartPM2App(args)
			if err != nil {
				log.Printf("[PM2] Error reiniciando: %v", err)
				s.reply(chatID, s.EscapeMD2(fmt.Sprintf("❌ Error al reiniciar '%s'.", args)))
				continue
			}
			s.reply(chatID, s.EscapeMD2(resultMsg))

		case "dockerstatus":
			statusMsg, err := commands.GetDockerStatus()
			if err != nil {
				log.Printf("[Docker] Error: %v", err)
				s.reply(chatID, s.EscapeMD2("Error consultando Docker. Asegúrate de tener configurado /var/run/docker.sock en los volúmenes."))
				continue
			}
			s.reply(chatID, s.EscapeMD2("🐳 *Estado de Docker:*\n\n")+s.EscapeMD2(statusMsg))

		case "dockerrestart", "dockerstop", "dockerstart":
			args := strings.TrimSpace(update.Message.CommandArguments())
			if args == "" {
				s.reply(chatID, s.EscapeMD2(fmt.Sprintf("⚠️ Debes especificar el nombre del contenedor. Ejemplo: /%s nginx", command)))
				continue
			}

			// Extraer la acción del comando (ej. dockerrestart -> restart)
			action := strings.TrimPrefix(command, "docker")
			
			resultMsg, err := commands.ManageDockerContainer(action, args)
			if err != nil {
				log.Printf("[Docker] Error en %s: %v", command, err)
				s.reply(chatID, s.EscapeMD2(fmt.Sprintf("❌ Error al intentar aplicar '%s' al contenedor '%s'.", action, args)))
				continue
			}
			s.reply(chatID, s.EscapeMD2(resultMsg))
		}
	}
}
