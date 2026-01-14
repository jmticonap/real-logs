# RealLogs: Kubernetes Pod Log Collector

**RealLogs** is a powerful tool designed to collect logs from Kubernetes pods. It can capture logs in real-time, from a specific directory, within a time range, or all at once. The collected logs are stored in a SQLite database, making it ideal for scenarios like stress testing where pods can restart or replicate quickly.

## 🧩 Key Features

- **Real-time Log Collection**: Stream logs as they happen.
- **Directory-based Collection**: Process logs from a specified directory.
- **Time-based Collection**: Gather logs within a specific time window.
- **Full Log Dump**: Download all available logs from pods.
- **Automatic Pod Detection**: Detects pod restarts and new pod creations to ensure continuous logging.
- **Organized Storage**: Saves logs in a structured SQLite database.
- **Flexible Configuration**: Use a `config.json` file or command-line flags for setup.

## 🛠️ Requirements

- Go 1.18+
- Access to a Kubernetes cluster configured via:
  - `InClusterConfig` (inside the cluster)
  - `~/.kube/config` (outside the cluster)
- Permissions to access pods and read logs.

## 🚀 Getting Started

### Dependency Installation

```sh
go mod tidy
```

### Configuration File

The `config.json` file is used to configure the application. Here is the expected structure:

```json
{
  "namespace": "your-namespace",
  "labelSelector": "app=your-app",
  "logDirectory": "./logs",
  "startTime": "14:00",
  "endTime": "15:00"
}
```

- `namespace`: The Kubernetes namespace to target.
- `labelSelector`: The label to filter pods by.
- `logDirectory`: The directory to store log files.
- `startTime` and `endTime`: The time range for the `btimes` flow.

### Makefile Execution

- **Run in development mode**:
  ```sh
  make run-dev
  ```
- **Build the application**:
  ```sh
  make build
  ```

## ⚙️ Usage and Flows

**RealLogs** offers four main flows, which can be selected using the `-flow` flag.

### 1. `realtime` Flow

This flow captures logs in real-time. It automatically handles pod restarts and new pod creations.

```sh
./reallogs -flow=realtime -dir=./log-1 -srv=se-core-charge
```

- `-dir`: (Optional) Specifies the directory to save logs. Defaults to the `logDirectory` in `config.json`.
- `-srv`: (Optional) The service name to filter pods. Use `all` to get logs from all pods in the namespace.

### 2. `fromdir` Flow

This flow processes log files from a specified directory and stores them in the SQLite database.

```sh
./reallogs -flow=fromdir -dir=./log-1
```

- `-dir`: (Required) The directory containing the log files to process.

### 3. `btimes` Flow

This flow collects logs within a specific time range. The start and end times can be provided via flags or the `config.json` file.

```sh
./reallogs -flow=btimes -start=14:00 -end=15:00
```

- `-start`: The start time in `HH:MM` or `YYYY-MM-DDTHH:MM` format.
- `-end`: (Optional) The end time in `HH:MM` or `YYYY-MM-DDTHH:MM` format. If not provided, the current time is used.

### 4. `fulllog` Flow

This flow downloads all available logs from the pods that match the selector.

```sh
./reallogs -flow=fulllog -dir=./full-logs
```

- `-dir`: (Optional) The directory to save the full logs.

## 📊 Command-line Flags

- `-flow`: Defines the execution flow (`realtime`, `fromdir`, `btimes`, `fulllog`).
- `-dir`: Specifies the target directory for logs.
- `-ns`: Overrides the namespace from `config.json`.
- `-srv`: Filters pods by a service name (`all` for all pods).
- `-start`: The start time for the `btimes` flow.
- `-end`: The end time for the `btimes` flow.
- `-batchs`: The batch size for database insertions.
- `-logperform`: Enables processing of performance logs.
- `-cpuprofile`: Specifies a file to write a CPU profile to.
- `-memprofile`: Specifies a file to write a memory profile to.

## 🗃️ SQLite Database

**RealLogs** creates a `log.db` file in the specified log directory. This database contains two tables:

- `general_logs`: Stores all log entries.
- `performance_logs`: Stores performance-related log data if `-logperform` is enabled.

## 🧠 Memory Profile

In a test with a 245MB data volume, the application's memory usage was approximately 1104MB. This is important to consider for resource planning.