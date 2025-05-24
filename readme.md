# RealLogs
## Kubernetes Pod Log Collector

Este programa permite recolectar desde una carpeta los logs descargados y en tiempo real de todos los pods que coincidan con un `labelSelector` dentro de un `namespace` específico en un clúster de Kubernetes. Los cuales serán ingresados en la base de datos Sqlite. Está diseñado especialmente para escenarios como **pruebas de estrés**, donde los pods pueden reiniciarse o replicarse rápidamente.

## 🧩 Características

- Recolecta logs en tiempo real (`stream`).
- Detecta cuando un pod se reinicia y reanuda la descarga de logs.
- Crea nuevos archivos de log si se crean nuevos pods.
- Guarda todos los logs en archivos separados, uno por pod.
- Usa un archivo `config.json` para su configuración.
- Crea automáticamente el directorio de logs si no existe.

## 🛠️ Requisitos

- Go 1.18+
- Acceso a un clúster de Kubernetes configurado vía:
  - `InClusterConfig` (dentro del clúster) o
  - `~/.kube/config` (fuera del clúster)
- Permisos para acceder a los pods y leer logs.

## Descarga de dependencias
```sh
go mod tidy
```
## 📁 Estructura esperada del archivo `config.json`

```json
{
  "namespace": "ecommerce-qas",
  "labelSelector": "app=se-core-charge",
  "logDirectory": "./logs"
}
```

## Ejecución con Makefile
- Ejecutar en modo desarrollo
```sh
make run-dev
```
- Ejecutar el build, salida (reallogs)
```sh
make build
```