# Valkiria Monitor

Un microservicio de alto rendimiento escrito en **Go** diseñado para monitorizar servidores Linux (VPS, Droplets). Supervisa proactivamente el uso de RAM, CPU y Disco, emitiendo alertas críticas a canales de Telegram y Discord.

Además, permite consultar el estado del servidor a demanda mediante un bot de Telegram.

## Características

- 📊 **Métricas Precisas:** Monitoriza RAM, CPU, Disco y Uptime mediante `gopsutil`.
- 🚨 **Alertas Proactivas:** Envía alertas automáticas cuando los recursos superan umbrales críticos (por defecto: RAM > 85%).
- 💬 **Bot Interactivo (Telegram):** Responde al comando `/statusvps` con un resumen detallado y amigable (soporta emojis y MarkdownV2).
- ⚡ **Ultra Ligero:** Escrito en Go, compilado estáticamente. Prácticamente no consume recursos en segundo plano.

---

## 🛠 Instalación y Configuración

### 1. Prerrequisitos
- **Go 1.18** o superior instalado.
- Un Bot de Telegram (obtenido desde [@BotFather](https://t.me/BotFather)) y tu ID de administrador.
- Un Webhook de Discord para enviar notificaciones a un canal.

### 2. Configuración del Entorno
Clona este repositorio o copia los archivos. Luego, en la raíz del proyecto, crea un archivo llamado `.env` con la siguiente estructura:

```env
TELEGRAM_BOT_TOKEN=tu_token_aqui
TELEGRAM_ADMIN_CHAT_ID=tu_chat_id_numerico
DISCORD_WEBHOOK_URL=tu_webhook_aqui
```

### 3. Compilación y Ejecución Manual
Para descargar las dependencias y compilar el proyecto:

```bash
go mod tidy
go build -o valkiria cmd/valkiria/main.go
```

Para probarlo manualmente y verificar que se conecta correctamente:
```bash
./valkiria
```

---

## 🚀 Despliegue en VPS Linux (Recomendado)

La mejor manera de ejecutar Valkiria Monitor en un servidor en producción es utilizar un servicio de **Systemd**.

1. **Mueve el ejecutable a un lugar seguro:**
   ```bash
   sudo mkdir -p /opt/valkiria-monitor
   sudo mv valkiria /opt/valkiria-monitor/
   # Asegúrate de colocar también tu archivo .env en /opt/valkiria-monitor/
   ```

2. **Crea el archivo del servicio:**
   ```bash
   sudo nano /etc/systemd/system/valkiria.service
   ```
   *Pega el siguiente contenido:*
   ```ini
   [Unit]
   Description=Valkiria System Monitor
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/opt/valkiria-monitor
   ExecStart=/opt/valkiria-monitor/valkiria
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```
   *(Nota: Se usa el usuario `root` para que el script pueda leer métricas profundas de las particiones del sistema sin problemas de permisos).*

3. **Habilita y arranca el servicio:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable valkiria
   sudo systemctl start valkiria
   ```

4. **Revisar los Logs:**
   Si necesitas ver qué está sucediendo o depurar algún error:
   ```bash
   sudo journalctl -u valkiria -f
   ```

---

## 🤖 Uso
Una vez que la aplicación esté corriendo, simplemente ve al chat de tu bot en Telegram y envía:

`/statusvps`

El bot responderá instantáneamente con el uso de CPU, RAM, Disco y el tiempo que lleva encendido el servidor.

---

## Licencia
Este proyecto se distribuye bajo la licencia **MIT**. Eres libre de utilizarlo, modificarlo y distribuirlo de forma personal o comercial.
