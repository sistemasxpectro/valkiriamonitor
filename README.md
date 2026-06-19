# Valkiria Monitor

Un microservicio de alto rendimiento escrito en **Go** diseñado para monitorizar servidores Linux (VPS, Droplets). Supervisa proactivamente el uso de RAM, CPU y Disco, emitiendo alertas críticas a canales de Telegram y Discord.

Además, permite consultar el estado del servidor a demanda mediante un bot de Telegram.

## Características

- 📊 **Métricas Precisas:** Monitoriza RAM, CPU, Disco y Uptime mediante `gopsutil`.
- 🚨 **Alertas Proactivas:** Envía alertas automáticas cuando los recursos superan umbrales críticos (por defecto: RAM > 85%).
- 💬 **Bot Interactivo (Telegram):** Responde al comando `/statusvps` con un resumen detallado y amigable (soporta emojis y MarkdownV2).
- 🔄 **Integración con PM2:** Consulta el estado de las aplicaciones con `/pm2status`, reinícialas remotamente con `/pm2restart`, y recibe alertas instantáneas en Telegram si una aplicación Node.js se cae inesperadamente.
- 🛡️ **Seguridad Estricta:** Comandos administrativos restringidos exclusivamente a tu ID de Telegram y webhooks protegidos por token local.
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
INTERNAL_API_TOKEN=tu_token_secreto_aqui
SERVER_NAME=TU_SERVIDOR
```

*Nota: La variable `SERVER_NAME` es opcional pero muy útil si tienes múltiples servidores conectados al mismo grupo de Telegram. Permite identificar de qué servidor provienen las alertas (ej. "Estado del TIQUEO" en lugar de "Estado del VPS").*

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

5. **Actualizar y Reiniciar el servicio:**
   Si subes una nueva versión del código a tu servidor o modificas el archivo `.env`, debes recompilar el binario y reiniciar el servicio:
   ```bash
   # Recompilar y reemplazar el binario
   go build -o valkiria cmd/valkiria/main.go
   sudo mv valkiria /opt/valkiria-monitor/
   
   # Reiniciar el servicio
   sudo systemctl restart valkiria
   
   # (Opcional) Verificar que arrancó bien
   sudo systemctl status valkiria
   ```

---

## 🐳 Despliegue con Docker y Docker Compose (Alternativa)

Si prefieres mantener tu servidor limpio y utilizar contenedores, puedes desplegar Valkiria Monitor utilizando Docker Compose. Se ha configurado para leer correctamente las métricas reales del host.

1. **Asegúrate de tener Docker y Docker Compose instalados.**
2. **Clona el repositorio** y crea el archivo `.env` como se indica en el paso anterior.
3. **Inicia el contenedor en segundo plano:**
   ```bash
   docker-compose up -d --build
   ```

Con esto, el servicio quedará corriendo, se reiniciará automáticamente si el servidor se apaga (`restart: unless-stopped`) y tendrá acceso de solo lectura a `/proc` y `/sys` del host para brindar métricas exactas.

Para ver los logs del contenedor:
```bash
docker-compose logs -f valkiria-monitor
```

---

## 🤖 Uso
Una vez que la aplicación esté corriendo, simplemente ve al chat de tu bot en Telegram y envía:

- `/statusvps`: El bot responderá instantáneamente con el uso de CPU, RAM, Disco y el tiempo que lleva encendido el servidor.
- `/pm2status`: Obtiene la lista de aplicaciones gestionadas por PM2, su uso de CPU, memoria y si están online/offline.
- `/pm2restart <app_name|all>`: Reinicia una o todas las aplicaciones en PM2 remotamente.

*(Nota: Cualquier comando enviado por un usuario que no sea el `TELEGRAM_ADMIN_CHAT_ID` será ignorado por seguridad).*

---

## 🌉 Puente PM2 (Alertas Reactivas)
Si deseas recibir alertas inmediatas cuando una aplicación de Node gestionada por PM2 haga "crash", debes arrancar el script puente incluido en la carpeta `scripts/`.

En tu servidor, donde esté alojado Valkiria Monitor:
1. Copia el archivo `scripts/pm2-listener.js`.
2. Ejecútalo pasándole el token interno que configuraste en tu `.env`:
```bash
INTERNAL_API_TOKEN=tu_token_secreto_aqui pm2 start pm2-listener.js --name "valkiria-pm2-bridge" --max-memory-restart 50M
pm2 save
```
Este script consumirá menos de 35 MB de RAM y actuará como puente silencioso notificando caídas de forma segura.

---

## Licencia
Este proyecto se distribuye bajo la licencia **MIT**. Eres libre de utilizarlo, modificarlo y distribuirlo de forma personal o comercial.
