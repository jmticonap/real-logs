package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/db"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	"github.com/jmticonap/real-logs/infrastructure/service"
	"github.com/jmticonap/real-logs/utils"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	// Flags
	flow := flag.String("flow", domain.RealTime, "Define que flujo se utiliza")
	dir := flag.String("dir", "", "Define el path del directorio objetivo")
	srvName := flag.String("srv", "all", "Define el nombre del servicio con el cual se filtran los pods")
	startFlag := flag.String("start", "", "Hora de inicio en formato HH:MM (opcional, también puede ir en config)")
	endFlag := flag.String("end", "", "Hora de fin en formato HH:MM (opcional, también puede ir en config)")
	batchSize := flag.Int("batchs", 50, "Largo del batch para las inserciones")
	logPerform := flag.Bool("logperform", false, "Define si se procesan los datos del log de performance")
	flag.Parse()

	// pprof for CPU
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

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

		srvCtx := context.WithValue(
			ctx,
			domain.CtxKeyType("srvName"),
			*srvName,
		)
		dirCtx := context.WithValue(
			srvCtx,
			domain.CtxKeyType("dir"),
			*dir,
		)
		logPerformCtx := context.WithValue(
			dirCtx,
			domain.CtxKeyType("logPerform"),
			*logPerform,
		)
		service.RealTimeProcess(logPerformCtx, cfg)

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
		logPerformCtx := context.WithValue(
			ctx,
			domain.CtxKeyType("logPerform"),
			*logPerform,
		)
		service.FromDir(logPerformCtx, targetDir)
	}

	// pprof for Memory
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC()

		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
