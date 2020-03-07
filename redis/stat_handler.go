package redis

type StatHandlerConfig struct {
	Redis RedisConfig `json:"redis"`
}

type StatHandler struct {
	Redis *Redis
}

func NewStatHandler(config *StatHandlerConfig) *StatHandler {
	result := &StatHandler{
		Redis: NewRedis(&config.Redis),
	}

	return result
}

func (sh *StatHandler) ShutDown() {

}
