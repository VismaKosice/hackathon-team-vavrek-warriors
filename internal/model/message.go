package model

type CalculationMessage struct {
	ID      int    `json:"id"`
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	LevelCritical = "CRITICAL"
	LevelWarning  = "WARNING"
)
