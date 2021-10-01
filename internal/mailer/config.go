package mailer

type Config struct {
	Address string `json:"address"`
	FromName string `json:"from_name"`
	FromEmail string `json:"from_email"`
}
