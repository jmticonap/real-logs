package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/db"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	"github.com/jmticonap/real-logs/infrastructure/service"
	"github.com/jmticonap/real-logs/utils"
)

func main() {
	// Flags
	flow := flag.String("flow", domain.RealTime, "Define que flujo se utiliza")
	dir := flag.String("dir", "", "Define el path del directorio objetivo")
	startFlag := flag.String("start", "", "Hora de inicio en formato HH:MM (opcional, también puede ir en config)")
	endFlag := flag.String("end", "", "Hora de fin en formato HH:MM (opcional, también puede ir en config)")
	batchSize := flag.Int("batchs", 50, "Largo del batch para las inserciones")
	// logType := flag.String("log-type", "", "")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Manejo de señales para cerrar la app con CTRL+C
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()

	// Leer archivo de configuración
	cfg, err := service.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if dir != nil && *dir != "" {
		db.OpenDb(domain.StrObject{"dir": *dir})
	} else {
		db.OpenDb(domain.StrObject{"dir": cfg.LogDirectory})
	}
	log.Println("DB Opened")

	errLogDir := utils.EnsureDir(cfg.LogDirectory)
	if errLogDir != nil {
		log.Fatalf("Error creating log dir: %v", errLogDir)
	}

	repository.StartGeneralLogWorker(ctx, *batchSize)
	repository.StartWriterWorker(ctx, *batchSize)

	switch *flow {
	case domain.RealTime:
		fmt.Println("Flujo RealTime")
		log.Println("Download logs in real time.")
		service.RealTimeProcess(ctx, cfg)

	case domain.BetweenTimes:
		fmt.Println("Flujo BetweenTimes")
		log.Println("Download logs between times.")
		var startTimeStr, endTimeStr string

		if *startFlag != "" {
			startTimeStr = *startFlag
			endTimeStr = *endFlag
		} else if cfg.StartTime != "" {
			startTimeStr = cfg.StartTime
			endTimeStr = cfg.EndTime
		} else {
			log.Fatal("Debes proporcionar -start al menos y/o -end o definirlos en config.json")
		}

		startTime, err := utils.ParseHour(startTimeStr)
		if err != nil {
			log.Fatalf("startTime inválido: %v", err)
		}

		var endTime time.Time
		if endTimeStr != "" {
			endTime, err = utils.ParseHour(endTimeStr)
			if err != nil {
				log.Fatalf("endTime inválido: %v", err)
			}
		} else {
			endTime = time.Now()
		}

		if endTime.Before(startTime) {
			log.Fatal("endTime no puede ser anterior a startTime")
		}

		log.Printf(
			"Descargando logs entre %s y %s...\n",
			startTime.Format("15:04"),
			endTime.Format("15:04"),
		)
		service.BetweenTimesProcess(ctx, cfg, startTime, endTime)

	case domain.FromDir:
		var targetDir string
		if dir != nil && *dir != "" {
			fmt.Printf("flag| dir=%s\n", *dir)
			targetDir = *dir
		} else if cfg.LogDirectory != "" {
			fmt.Printf("Config: %s", cfg.LogDirectory)
			targetDir = cfg.LogDirectory
		} else {
			log.Fatalln("No hay un directorio destino configurado.")
		}
		service.FromDir(ctx, targetDir)
	}
}
