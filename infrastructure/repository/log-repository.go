package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/db"
	"github.com/jmticonap/real-logs/utils"
)

var logChan = make(chan domain.LogChanDataType, 1000)

func LogChanPush(
	logData domain.LogType,
	performanceData []domain.PerformanceType,
) {
	t, err := time.Parse("2006-01-02T15:04:05.000-07:00", logData.Timestamp)
	if err != nil {
		log.Panicf("Error parsing: %s", err)
		return
	}

	var params []any = []any{}
	for _, perform := range performanceData {
		params := append(
			params,
			logData.TraceId,
			perform.Method,
			perform.Exectime,
			perform.MemoryUsage,
			t.Format(time.RFC3339Nano),
		)

		logChan <- domain.LogChanDataType{
			Params: params,
		}
		params = params[:0]
	}
}

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
				} else {
					fmt.Printf("\rSaved data: %s", LogQueryData.Params)
				}
			}
		}
	}()
}

func SaveLog(ctx context.Context, line string) {
	log, err := utils.GetLogItem(line)
	if err != nil {
		return
	}
	logPerformanceInfo, err := utils.GetPerformanceLogInfo(log)
	if err != nil {
		return
	}
	LogChanPush(log, logPerformanceInfo)
}
