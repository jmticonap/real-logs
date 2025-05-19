package domain

import "regexp"

var (
	TimeRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2}))\]`),                 // Ej: [2025-05-15T17:22:59-0500]
		regexp.MustCompile(`"timestamp"\s*:\s*"(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2}))"`), // Ej: "timestamp":"2025-05-15T17:22:59.820-05:00"
	}
)

const (
	RealTime     string = "realtime"
	BetweenTimes string = "btimes"
)
