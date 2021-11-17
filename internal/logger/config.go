package logger

import (
	"fmt"
	"github.com/hawell/z42/configs"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level       string `json:"level"`
	Destination string `json:"destination"`
}

func DefaultConfig() Config {
	return Config{
		Level:       "info",
		Destination: "stdout",
	}
}

func (c Config) Verify() {
	fmt.Println("checking log...")
	msg := fmt.Sprintf("checking log level: %s", c.Level)
	level := zapcore.InfoLevel
	err := level.UnmarshalText([]byte(c.Level))
	configs.PrintResult(msg, err)
	msg = fmt.Sprintf("checking whether %s is available", c.Destination)
	_, err = NewLogger(&c)
	configs.PrintResult(msg, err)
}
