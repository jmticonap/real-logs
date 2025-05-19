package repository

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/db"
)

var logChan = make(chan domain.LogChanDataType, 1000)

func StartWriterWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Finalizando SQLite writer")
				return

			case LogQueryData := <-logChan:
				db := db.OpenDb()
				query := `
					INSERT INTO performance_logs 
					(trace_id, method, exectime, memory_mb, timestamp)
					VALUES (?, ?, ?, ?, ?)
				`
				_, err := db.ExecContext(ctx, query, LogQueryData.Params...)
				if err != nil {
					log.Printf("Error inserting data: %s", err)
				}
			}
		}
	}()
}

func SaveLog(ctx context.Context, line string) {
	var params []any = []any{}

	// re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}-\d{2}:\d{2}`)
	// match := re.FindStringSubmatch(line)
	// if match == nil {
	// 	log.Panicln("No se encontró fecha")
	// 	return
	// }
	var logData domain.LogType
	if err := json.Unmarshal([]byte(line), &logData); err != nil {
		log.Panicf("Unmarshelling error: %s", err)
	}

	performanceData, err := getPerformanceData(logData.Msg)
	if err != nil {
		log.Panicf("Error reading performance data: %s", err)
		return
	}

	t, err := time.Parse("2006-01-02T15:04:05.000-07:00", logData.Timestamp)
	if err != nil {
		log.Panicf("Error parsing: %s", err)
		return
	}

	params = append(
		params,
		logData.TraceId,
		performanceData.Method,
		performanceData.Exectime,
		performanceData.MemoryUsage,
		t.Format(time.RFC3339Nano),
	)

	logChan <- domain.LogChanDataType{
		Params: params,
	}
}

func getPerformanceData(line string) (domain.PerformanceType, error) {
	// Parsear el JSON externo
	var logData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &logData); err != nil {
		return domain.PerformanceType{}, err
	}

	// Extraer el campo "msg"
	rawMsg := logData["msg"].(string)

	// Limpiar el string del campo "msg"
	// - Reemplazar comillas simples por comillas dobles
	// - Reemplazar claves sin comillas por claves con comillas
	clean := strings.ReplaceAll(rawMsg, "'", `"`)
	re := regexp.MustCompile(`(?m)(\s*)(\w+):`)
	clean = re.ReplaceAllString(clean, `$1"$2":`)

	// Parsear el string limpio como JSON
	var msgData domain.PerformanceType
	if err := json.Unmarshal([]byte(clean), &msgData); err != nil {
		return domain.PerformanceType{}, err
	}

	return msgData, nil
}
