package logger

type Config struct {
	Level       string `json:"level"`
	Destination string `json:"destination"`
}
