package mailer

type Config struct {
	Address       string `json:"address"`
	FromEmail     string `json:"from_email"`
	WebServer     string `json:"web_server"`
	ApiServer     string `json:"api_server"`
	NameServer    string `json:"name_server"`
	HtmlTemplates string `json:"html_templates"`
	Auth          Auth   `json:"auth"`
}

type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func DefaultConfig() Config {
	return Config{
		Address:       "127.0.0.1:25",
		FromEmail:     "noreply@z42.com",
		WebServer:     "www.z42.com",
		ApiServer:     "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "./templates/*.tmpl",
	}
}
