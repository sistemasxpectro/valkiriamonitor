package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"valkiria-monitor/internal/notifier"
)

type CrashPayload struct {
	App   string `json:"app"`
	Event string `json:"event"`
	Error string `json:"error"`
}

func StartInternalAPI(tg *notifier.TelegramService, discordURL string) {
	internalToken := os.Getenv("INTERNAL_API_TOKEN")
	if internalToken == "" {
		log.Println("[Advertencia] INTERNAL_API_TOKEN no configurado. La API interna no validará solicitudes.")
	}

	mux := http.NewServeMux()
	
	mux.HandleFunc("/pm2-alert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Validar token de autenticación interno
		if internalToken != "" && r.Header.Get("X-Valkiria-Token") != internalToken {
			log.Println("[Seguridad] Solicitud rechazada a /pm2-alert: token inválido")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var payload CrashPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Formateamos el mensaje para Telegram
		rawMsg := fmt.Sprintf("⚠️ *PM2 CRASH DETECTADO*\n\n*App:* %s\n*Evento:* %s\n*Detalle:* %s", 
			payload.App, payload.Event, payload.Error)

		// Disparamos notificaciones reactivas
		tg.SendCriticalAlert(rawMsg)
		notifier.SendDiscordAlert(discordURL, "🔴 **ALERTA DE PROCESO (PM2):**\n"+rawMsg)

		w.WriteHeader(http.StatusOK)
	})

	// Escuchamos en el puerto 8451 estrictamente en localhost (127.0.0.1)
	log.Println("API interna escuchando en 127.0.0.1:8451")
	go func() {
		if err := http.ListenAndServe("127.0.0.1:8451", mux); err != nil {
			log.Fatalf("Error en servidor interno: %v", err)
		}
	}()
}
