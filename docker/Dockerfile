# Usa una imagen reciente Go
FROM golang:1.24.2-alpine

# Instala Git si es necesario para dependencias
RUN apk add --no-cache git

# Crea el directorio de trabajo
WORKDIR /app

# Copia los archivos del proyecto
COPY . .

# Descarga dependencias
RUN go mod download

# Compila la aplicación
RUN go build -o reallogs main.go

# Expone el puerto que usa Fiber
EXPOSE 3000

# Comando de inicio
CMD ["./main"]
