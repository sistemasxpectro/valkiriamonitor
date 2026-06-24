# Add Docker Management to Valkiria Monitor

Este plan propone añadir la capacidad de gestionar contenedores Docker directamente desde el bot de Telegram de Valkiria Monitor, permitiéndote ver los estados y reiniciar, detener o iniciar contenedores.

## User Review Required

> [!IMPORTANT]
> Para que Valkiria Monitor pueda gestionar contenedores desde adentro de su propio contenedor, es necesario compartirle el socket de Docker del host (`/var/run/docker.sock`). Esto le da a la aplicación permisos para interactuar con el demonio de Docker. Como el bot está restringido solo a tu Chat ID de administrador, esto es seguro, pero es importante que lo sepas.

## Open Questions

> [!NOTE]
> 1. Añadiré una validación de seguridad para evitar que el bot se detenga o reinicie a sí mismo (el contenedor `valkiria-monitor`), ¿te parece bien?
> 2. ¿Quieres que el comando `/dockerstatus` muestre todos los contenedores (incluyendo los detenidos `docker ps -a`) o solo los que están corriendo actualmente (`docker ps`)? (Por defecto mostraré solo los que están corriendo para no saturar el chat).

## Proposed Changes

### Docker Configuration
#### [MODIFY] [Dockerfile](file:///c:/Desarrollo/ValkiriaMonitor/Dockerfile)
- Se instalará el paquete `docker-cli` en la imagen final de Alpine. Esto permite usar el comando `docker` nativamente a través de Go.

#### [MODIFY] [docker-compose.yml](file:///c:/Desarrollo/ValkiriaMonitor/docker-compose.yml)
- Se añadirá el volumen `- /var/run/docker.sock:/var/run/docker.sock` para conectar el cliente interno con el demonio del servidor.

### Go Application Logic
#### [NEW] [internal/commands/docker.go](file:///c:/Desarrollo/ValkiriaMonitor/internal/commands/docker.go)
- Creación de las funciones:
  - `GetDockerStatus()`: Ejecutará `docker ps --format '{{json .}}'` y parseará la salida para enviarla a Telegram.
  - `ManageDockerContainer(action, containerName string)`: Ejecutará `docker start|stop|restart <nombre>`.

#### [MODIFY] [internal/notifier/telegram.go](file:///c:/Desarrollo/ValkiriaMonitor/internal/notifier/telegram.go)
- Se añadirán los nuevos comandos al `switch` de Telegram:
  - `/dockerstatus`
  - `/dockerrestart <contenedor>`
  - `/dockerstop <contenedor>`
  - `/dockerstart <contenedor>`

#### [MODIFY] [README.md](file:///c:/Desarrollo/ValkiriaMonitor/README.md)
- Actualizar la documentación indicando los nuevos comandos disponibles.

## Verification Plan

### Manual Verification
- Te pediré que hagas un `git pull` y luego recrees el contenedor con `sudo docker-compose up -d --build`.
- Luego, desde Telegram, probarás mandar `/dockerstatus` y comprobarás si lista los contenedores activos.
