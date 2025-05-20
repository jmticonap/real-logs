package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	"github.com/jmticonap/real-logs/utils"
)

func FromDir(ctx context.Context, dirPath string) {
	fmt.Printf("DIR: %s\n", dirPath)
	paths, err := GetAllFilesRecursive(dirPath)
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
			logPerformanceInfo, err := getPerformanceLogInfo(log)
			if err != nil {
				continue
			}
			repository.LogChanPush(log, logPerformanceInfo)
		}
	}
}

func GetAllFilesRecursive(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func getPerformanceLogInfo(log domain.LogType) ([]domain.PerformanceType, error) {
	rawMsg := log.Msg
	clean := strings.ReplaceAll(rawMsg, "'", `"`)
	re := regexp.MustCompile(`(?m)(\s*)(\w+):`)
	clean = re.ReplaceAllString(clean, `$1"$2":`)
	var performanceLog domain.PerformanceLogType
	if err := json.Unmarshal([]byte(clean), &performanceLog); err != nil {
		return []domain.PerformanceType{}, err
	}

	return performanceLog.PerformanceInfo, nil
}
