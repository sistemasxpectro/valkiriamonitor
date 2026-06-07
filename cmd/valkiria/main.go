package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"valkiria-monitor/internal/metrics"
	"valkiria-monitor/internal/notifier"
	"valkiria-monitor/internal/server"
)

func main() {
	// Cargar archivo .env si existe
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró el archivo .env o no se pudo cargar, se usarán las variables del entorno del sistema")
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramAdminIDStr := os.Getenv("TELEGRAM_ADMIN_CHAT_ID")
	discordWebhook := os.Getenv("DISCORD_WEBHOOK_URL")

	if telegramToken == "" || telegramAdminIDStr == "" || discordWebhook == "" {
		log.Fatal("Faltan variables de entorno requeridas: TELEGRAM_BOT_TOKEN, TELEGRAM_ADMIN_CHAT_ID, DISCORD_WEBHOOK_URL")
	}

	telegramAdminID, err := strconv.ParseInt(telegramAdminIDStr, 10, 64)
	if err != nil {
		log.Fatalf("El ID del administrador de Telegram no es válido: %v", err)
	}

	telegramService, err := notifier.NewTelegramService(telegramToken, telegramAdminID)
	if err != nil {
		log.Fatalf("No se pudo inicializar el servicio de Telegram: %v", err)
	}

	log.Println("Valkiria Monitor iniciado... Escuchando comandos y evaluando métricas.")

	// Lanza una goroutine que se ejecute cada 5 minutos evaluando las métricas
	go func() {
		// Evaluamos inmediatamente al arrancar
		evaluateMetrics(discordWebhook, telegramService)

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			evaluateMetrics(discordWebhook, telegramService)
		}
	}()

	// Lanza la API interna para webhook local de PM2
	server.StartInternalAPI(telegramService, discordWebhook)

	// Bloquea el hilo principal ejecutando la escucha reactiva de comandos de Telegram
	telegramService.StartListening()
}

func evaluateMetrics(discordWebhook string, telegramService *notifier.TelegramService) {
	stats, err := metrics.GetStats()
	if err != nil {
		log.Printf("Error obteniendo métricas en el proceso de fondo: %v", err)
		return
	}

	// Si RAM > 85%, dispara alertas simultáneas por Discord y Telegram.
	// stats.DiskUsage > 90.0 || (Comentado temporalmente)
	if stats.RAMUsage > 85.0 {
		alertMsg := fmt.Sprintf(
			"🚨 *ALERTA DE SISTEMA* 🚨\n\n"+
				"🧠 *RAM:* %.2f%% (Uso Elevado)\n"+
				"⚙️ *CPU:* %.2f%%\n"+
				"💽 *Disco:* %.2f%%\n\n"+
				"⚠️ *Atención:* Se ha detectado un consumo crítico de memoria en el servidor.",
			stats.RAMUsage, stats.CPUUsage, stats.DiskUsage,
		)
		
		// Enviar por Discord
		if err := notifier.SendDiscordAlert(discordWebhook, alertMsg); err != nil {
			log.Printf("Error enviando alerta por Discord: %v", err)
		} else {
			log.Println("Alerta de Discord enviada correctamente.")
		}

		// Enviar por Telegram
		if err := telegramService.SendCriticalAlert(alertMsg); err != nil {
			log.Printf("Error enviando alerta por Telegram: %v", err)
		} else {
			log.Println("Alerta de Telegram enviada correctamente.")
		}
	}
}
