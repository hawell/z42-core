package healthcheck

type Config struct {
	Enable             bool `json:"enable"`
	MaxRequests        int  `json:"max_requests"`
	MaxPendingRequests int  `json:"max_pending_requests"`
	UpdateInterval     int  `json:"update_interval"`
	CheckInterval      int  `json:"check_interval"`
}

func DefaultConfig() Config {
	return Config{
		Enable:             false,
		MaxRequests:        10,
		MaxPendingRequests: 100,
		UpdateInterval:     600,
		CheckInterval:      600,
	}
}

func (_ Config) Verify() {
}
