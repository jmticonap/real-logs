package service

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	"github.com/jmticonap/real-logs/utils"
)

func FromDir(ctx context.Context, dirPath string) {
	logPerform := ctx.Value(domain.CtxKeyType("logPerform")).(bool)
	paths, err := utils.GetAllFilesRecursive(dirPath)
	if err != nil {
		log.Fatalf("Error reading dir: %s", err)
	}

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			log.Fatalf("Error openning file: %s", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			log, err := utils.GetLogItem(line)
			if err != nil {
				continue
			}
			repository.GeneralChanPush(log)

			if logPerform {
				logPerformanceInfo, err := utils.GetPerformanceLogInfo(log)
				if err != nil {
					continue
				}
				repository.LogChanPush(log, logPerformanceInfo)
			}
		}
	}
}
