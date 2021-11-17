package ratelimit

type Config struct {
	Enable    bool     `json:"enable"`
	Burst     int      `json:"burst"`
	Rate      int      `json:"rate"`
	WhiteList []string `json:"whitelist"`
	BlackList []string `json:"blacklist"`
}

func DefaultConfig() Config {
	return Config{
		Enable:    false,
		Rate:      60,
		Burst:     10,
		BlackList: []string{},
		WhiteList: []string{},
	}
}
