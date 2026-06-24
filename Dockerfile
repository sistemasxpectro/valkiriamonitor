# Etapa de construcción (Builder)
FROM golang:1.18-alpine AS builder

# Instalar dependencias necesarias para compilar gopsutil si es necesario
RUN apk add --no-cache git build-base

WORKDIR /app

# Copiar go.mod y go.sum (si existe) y descargar dependencias
COPY go.mod go.sum* ./
RUN go mod download

# Copiar el resto del código fuente
COPY . .

# Asegurar que las dependencias (go.sum) estén actualizadas
RUN go mod tidy

# Compilar el binario
RUN go build -o valkiria-monitor ./cmd/valkiria

# Etapa final (Producción)
FROM alpine:latest

WORKDIR /app

# Instalar certificados CA para las peticiones HTTPS a las APIs, y docker-cli para gestionar contenedores
RUN apk add --no-cache ca-certificates tzdata docker-cli

# Copiar el binario compilado desde la etapa builder
COPY --from=builder /app/valkiria-monitor .

# Variables de entorno por defecto para gopsutil
ENV HOST_PROC=/host/proc
ENV HOST_SYS=/host/sys
ENV HOST_ETC=/host/etc

# Exponer el puerto de la API interna
EXPOSE 8451

# Ejecutar el binario
CMD ["./valkiria-monitor"]
