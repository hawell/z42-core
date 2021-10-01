package mailer

type Config struct {
	Address string `json:"address"`
	FromName string `json:"from_name"`
	FromEmail string `json:"from_email"`
	WebServer    string  `json:"web_server"`
	ApiServer   string `json:"api_server"`
	NameServer   string `json:"name_server"`
	HtmlTemplates string `json:"html_templates"`
}
