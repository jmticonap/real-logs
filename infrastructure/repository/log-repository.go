package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/db"
	"github.com/jmticonap/real-logs/utils"
)

var logChan = make(chan domain.LogChanDataType, 1000)
var generalLogChan = make(chan domain.LogType, 1000)

func GeneralChanPush(logData domain.LogType) {
	generalLogChan <- logData
}

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
		params = append(
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

func StartWriterWorker(ctx context.Context, batchSize int) {
	go func() {
		db := db.OpenDb(domain.StrObject{})
		var batch []domain.LogChanDataType
		for {
			select {
			case <-ctx.Done():
				if len(batch) > 0 {
					insertBatchPerformanceLog(ctx, db, &batch)
				}
				log.Println("Finalizando SQLite writer")
				return

			case LogQueryData := <-logChan:
				batch = append(batch, LogQueryData)

				if len(batch) >= batchSize {
					insertBatchPerformanceLog(ctx, db, &batch)
				}
			}
		}
	}()
}

func StartGeneralLogWorker(ctx context.Context, batchSize int) {
	go func() {
		db := db.OpenDb(domain.StrObject{})
		var batch []domain.LogType
		for {
			select {
			case <-ctx.Done():
				if len(batch) > 0 {
					insertBatchGeneralLog(ctx, db, &batch)
				}
				log.Println("Finalizando general log SQLite writer")
				return

			case logData := <-generalLogChan:
				batch = append(batch, logData)

				if len(batch) >= batchSize {
					insertBatchGeneralLog(ctx, db, &batch)
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
	GeneralChanPush(log)
	logPerformanceInfo, err := utils.GetPerformanceLogInfo(log)
	if err != nil {
		return
	}
	LogChanPush(log, logPerformanceInfo)
}

func insertBatchPerformanceLog(
	ctx context.Context,
	db *sql.DB,
	batch *[]domain.LogChanDataType,
) {

	query := `
		INSERT INTO performance_logs 
		(trace_id, method, exectime, memory_mb, timestamp)
		VALUES 
	`
	queryValues := []string{}
	params := []any{}
	for _, log := range *batch {
		params = append(params, log.Params...)
		queryValues = append(queryValues, "(?, ?, ?, ?, ?)")
	}
	query += strings.Join(queryValues, ", ")

	_, err := db.ExecContext(
		ctx,
		query,
		params...,
	)
	if err != nil {
		log.Printf("Error inserting performance log data: %s", err)
	} else {
		fmt.Printf("\r[Performance] Saved data: BatchSize=%d", len(*batch))
	}
	*batch = (*batch)[:0]
}

func insertBatchGeneralLog(
	ctx context.Context,
	db *sql.DB,
	batch *[]domain.LogType,
) {

	query := `
		INSERT INTO general_logs
		(level, timestamp, hostname, trace_id, span_id, parent_id, msg)
		VALUES 
	`
	queryValues := []string{}
	params := []any{}
	for _, log := range *batch {
		params = append(
			params,
			log.Level,
			log.Timestamp,
			log.Hostname,
			log.TraceId,
			log.SpanId,
			log.ParentId,
			log.Msg,
		)
		queryValues = append(queryValues, "(?, ?, ?, ?, ?, ?, ?)")
	}
	query += strings.Join(queryValues, ", ")

	_, err := db.ExecContext(
		ctx,
		query,
		params...,
	)
	if err != nil {
		log.Printf("Error inserting general log data: %s", err)
	} else {
		fmt.Printf("\r[General] Saved data: BatchSize=%d", len(*batch))
	}
	*batch = (*batch)[:0]
}
