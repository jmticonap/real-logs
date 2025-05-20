package domain

type Config struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector"`
	LogDirectory  string `json:"logDirectory"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
}

type LogChanDataType struct {
	Params []any
}

type LogType struct {
	Level     string `json:"level"`
	Timestamp string `json:"timestamp"`
	Pid       int    `json:"pid"`
	Hostname  string `json:"hostname"`
	TraceId   string `json:"traceId"`
	SpanId    string `json:"spanId"`
	ParentId  string `json:"parentId"`
	Msg       string `json:"msg"`
}

type PerformanceLogType struct {
	Title           string            `json:"title"`
	PerformanceInfo []PerformanceType `json:"performanceInfo"`
}

type PerformanceType struct {
	Exectime    float32 `json:"exectime"`
	Origin      string  `json:"origin"`
	Method      string  `json:"method"`
	MemoryUsage string  `json:"memoryUsage"`
	Percentage  string  `json:"percentage"`
}
