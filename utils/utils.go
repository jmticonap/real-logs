package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jmticonap/real-logs/domain"
)

func EnsureDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// No existe, crear carpeta
		return os.MkdirAll(path, 0755)
	}
	if err != nil {
		// Otro error al intentar acceder
		return err
	}
	if !info.IsDir() {
		// Existe pero no es directorio, error
		return fmt.Errorf("%s ya existe pero no es un directorio", path)
	}
	// Existe y es directorio, todo ok
	return nil
}

func ParseHour(h string) (time.Time, error) {
	result := time.Time{}
	var err error = nil

	var parsed time.Time
	if parsed, err = time.Parse("15:04", h); err == nil {
		now := time.Now()

		// Combina la fecha de hoy con la hora proporcionada
		result = time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			parsed.Hour(),
			parsed.Minute(),
			0,
			0,
			now.Location(),
		)
	} else if parsed, err = time.Parse("2006-01-02T15:04", h); err == nil {
		result = parsed
	}

	return result, err
}

func GetLogItem(line string) (domain.LogType, error) {
	var log domain.LogType
	if err := json.Unmarshal([]byte(line), &log); err != nil {
		return domain.LogType{}, err
	}

	return log, nil
}
