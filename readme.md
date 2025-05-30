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
  "namespace": "ecommerce-dev",
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

## Flags
En la ejecución los valores que provienen del `config.json` siempre será la segunda opción.
- flow: Define el flujo que utiliza.
  - realtime: se guardan los logs en tiempo real y toman reintentos de lectura si el pod se reinicia.
  - fromdir: Define que a partir de un directorio con archivos de logs se leerán y se guardará toda la información en json en una base de datos Sqlite.
  - Ejemplo:
    ```sh
    ./reallogs -flow=realtime -dir=./log-1 -srv=se-core-charge
    ```
    Nota: Descarga los logs en tiempo real y los guarda en la ruta relativa "./log-1". En `-srv` puede asignar el valor `all` para obtener los logs de todos los pods dentro del namespace.

    ```sh
    ./reallogs -flow=fromdir -dir=./log-1
    ```
    Nota: Carga la información de los logs en formato json que encuentre en "./log-1" en una base de datos Sqlite

## Perfil de memoria actual
Para una prueba con un volumen de datos de 245Mb se tiene un resultante en memoria de 1104Mb. 