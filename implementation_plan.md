La propuesta de integración es excelente y muy necesaria. Al gestionar los procesos de Node para el backend de Tiqueo y tus otros proyectos en el Droplet, tener control de PM2 desde el teléfono te ahorrará muchísimo tiempo.Aquí tienes mis respuestas como Arquitecto DevOps a las preguntas abiertas y cómo proceder con la implementación:Decisiones de ArquitecturaSeguridad (Validación del Admin): Es absolutamente obligatorio. Modificaremos el bucle de escucha en Telegram para que cualquier Chat.ID que no coincida con el tuyo sea ignorado silenciosamente.Comando /reboot (Reinicio del Servidor): Descartado. Por principios de menor privilegio (Principio de PoLP), Valkiria Monitor no debe ejecutarse como root si no es estrictamente necesario, y darle capacidad de reiniciar el sistema operativo abre un vector de ataque peligroso. Si el VPS sufre un pánico del kernel o se congela a nivel de I/O, el bot de Telegram de todas formas no responderá. Para reinicios duros, es mejor depender de la consola nativa del proveedor de la nube.Ruta de PM2: Intentaremos llamarlo de forma global (pm2). Sin embargo, como los servicios administrados por systemd a veces tienen un $PATH muy limitado (especialmente si instalaste Node vía NVM), existe la posibilidad de que no lo encuentre. Empezaremos con el comando global y, si falla en producción, lo cambiaremos por la ruta absoluta (ej. /home/usuario/.nvm/versions/node/vX.X.X/bin/pm2).Instrucciones de Implementación (Código)Puedes pasarle este contexto a Antigravity o implementarlo directamente.1. Nuevo Archivo: internal/commands/pm2.goEste archivo se encargará de ejecutar los comandos en la terminal y parsear el JSON que devuelve PM2.Gopackage commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

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
	cmd := exec.Command("pm2", "restart", appName)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fallo al reiniciar '%s': %w", appName, err)
	}
	return fmt.Sprintf("✅ Proceso '%s' reiniciado correctamente\\.", appName), nil
}
2. Modificación: internal/notifier/telegram.goActualizamos el método StartListening() para incluir la validación de seguridad y los nuevos comandos.  Go// Añade esta importación arriba: "valkiria-monitor/internal/commands"

func (ts *TelegramService) StartListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := ts.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		// [SEGURIDAD] Ignorar cualquier mensaje que no venga del Administrador
		if update.Message.Chat.ID != ts.adminID {
			log.Printf("[Seguridad] Intento de acceso denegado del ChatID: %d", update.Message.Chat.ID)
			continue
		}

		command := update.Message.Command()
		chatID := update.Message.Chat.ID

		switch command {
		case "statusvps":
			stats, err := metrics.GetStats()
			if err != nil {
				ts.reply(chatID, "Error leyendo métricas del servidor\\.")
				continue
			}

			rawMsg := fmt.Sprintf(
				"🖥️ *CPU:* %.2f%%\n💾 *RAM:* %d MB / %d MB (%.2f%%)\n💽 *Disco:* %d GB / %d GB (%.2f%%)\n⏱️ *Uptime:* %d horas",
				stats.CPUUsage, stats.RAMUsedMB, stats.RAMTotalMB, stats.RAMUsagePct,
				stats.DiskUsedGB, stats.DiskTotalGB, stats.DiskUsagePct, stats.UptimeHours,
			)
			ts.reply(chatID, EscapeMD2(rawMsg))

		case "pm2status":
			statusMsg, err := commands.GetPM2Status()
			if err != nil {
				log.Printf("[PM2] Error: %v", err)
				ts.reply(chatID, "Error consultando PM2\\. Revisa los logs\\.")
				continue
			}
			ts.reply(chatID, EscapeMD2("📊 *Estado de PM2:*\n\n")+EscapeMD2(statusMsg))

		case "pm2restart":
			args := update.Message.CommandArguments()
			if args == "" {
				ts.reply(chatID, "⚠️ Debes especificar el nombre de la app o 'all'\\. Ejemplo: `/pm2restart api`")
				continue
			}
			
			resultMsg, err := commands.RestartPM2App(args)
			if err != nil {
				log.Printf("[PM2] Error reiniciando: %v", err)
				ts.reply(chatID, EscapeMD2(fmt.Sprintf("❌ Error al reiniciar '%s'\\.", args)))
				continue
			}
			ts.reply(chatID, EscapeMD2(resultMsg))
		}
	}
}
también reciba alertas pasivas del propio PM2 por si una aplicación de Node crashea repentinamente por un error de memoria

Para lograr esto sin comprometer la estabilidad del monitor, la forma más limpia y robusta arquitectónicamente es usar un patrón "Sidecar" local.

Conectar Go directamente al bus de eventos interno (RPC) de PM2 es ineficiente y propenso a romperse con cualquier actualización de la herramienta, ya que PM2 es un ecosistema puramente Node. En su lugar, abriremos un pequeño puerto HTTP privado en Valkiria Monitor y usaremos un script de Node.js ligerísimo que actúe como puente entre los eventos nativos de PM2 y Valkiria.

Aquí tienes la implementación exacta.

### 1. Valkiria Monitor: Servidor Interno de Recepción (Go)

Crearemos un nuevo paquete para exponer una API interna que escuche exclusivamente en `localhost`. Esto garantiza que nadie de afuera pueda inyectar alertas falsas.

**Crea el archivo `internal/server/local.go`:**

```go
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"valkiria-monitor/internal/notifier"
)

type CrashPayload struct {
	App   string `json:"app"`
	Event string `json:"event"`
	Error string `json:"error"`
}

func StartInternalAPI(tg *notifier.TelegramService, discordURL string) {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/pm2-alert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

```

**Actualiza `cmd/valkiria/main.go`:**
Añade la llamada a `StartInternalAPI` justo antes del motor reactivo de Telegram.

```go
// En las importaciones añade: "valkiria-monitor/internal/server"

// 3.5 Iniciar API local para alertas pasivas
server.StartInternalAPI(tgService, discordWebhook)

// 4. Motor Reactivo (Bloquea el main loop escuchando comandos)
tgService.StartListening()

```

---

### 2. Puente PM2 (Node.js)

En tu VPS, en cualquier directorio (por ejemplo, `/opt/valkiria-monitor/`), crea el siguiente script en JavaScript. Este aprovechará la API nativa de PM2 para suscribirse al bus de eventos y enviará un POST a Valkiria solo cuando ocurra un error crítico (excepciones no manejadas o bucles de reinicio).

**Crea el archivo `pm2-listener.js`:**

```javascript
const pm2 = require('pm2');

pm2.connect(function(err) {
    if (err) {
        console.error(err);
        process.exit(2);
    }

    pm2.launchBus(function(err, bus) {
        console.log('[PM2 Listener] Escuchando eventos críticos para Valkiria Monitor...');

        // Escucha excepciones no capturadas (Crashes de Node)
        bus.on('process:exception', function(data) {
            const appName = data.process.name;
            // Ignorar errores del propio listener para evitar bucles infinitos
            if (appName === 'pm2-listener') return; 

            const errorMsg = data.data.message || 'Excepción desconocida';
            sendAlert(appName, 'Exception (Crash)', errorMsg);
        });

        // Escucha cuando una app entra en bucle de reinicios (Restart loop)
        bus.on('process:event', function(data) {
            if (data.event === 'restart overlimit') {
                sendAlert(data.process.name, 'Restart Loop', 'La aplicación está crasheando repetidamente y PM2 dejó de reiniciarla.');
            }
        });
    });
});

function sendAlert(app, event, errorMsg) {
    // La API fetch está disponible nativamente en Node 18+
    fetch('http://127.0.0.1:8451/pm2-alert', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            app: app,
            event: event,
            error: errorMsg.substring(0, 200) // Truncamos por si el stacktrace es enorme
        })
    }).catch(err => {
        console.error('[PM2 Listener] Error conectando con Valkiria:', err.message);
    });
}

```

### 3. Puesta en Marcha

Una vez hayas compilado y reemplazado el binario de Valkiria en el servidor:

1. Reinicia el servicio de Valkiria:
`sudo systemctl restart valkiria-monitor`
2. Arranca el script de escucha utilizando el propio PM2, con un límite bajo de RAM ya que no hará nada el 99% del tiempo:
`pm2 start pm2-listener.js --name "valkiria-pm2-bridge" --max-memory-restart 50M`
3. Guarda el estado de PM2 para que el puente arranque en los reinicios del sistema:
`pm2 save`

Con esta configuración tienes lo mejor de ambos mundos: Valkiria es un binario inmutable en Go administrado por Systemd, y la captura de errores específicos de tus APIs recae en un micro-script nativo que le inyecta la información localmente.