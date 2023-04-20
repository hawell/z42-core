package mailer

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestSend(t *testing.T) {
	RegisterTestingT(t)
	m, err := NewSMTP(&Config{
		Address:       "127.0.0.1:25",
		FromEmail:     "noreply@zone-42.com",
		WebServer:     "www.z42.com",
		ApiServer:     "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "../../templates/*.tmpl",
	})
	Expect(err).To(BeNil())
	err = m.SendEMailVerification("arash", "arash@legioner", "test")
	Expect(err).To(BeNil())
}

func TestSendWithAuthGMail(t *testing.T) {
	RegisterTestingT(t)
	m, err := NewSMTP(&Config{
		Address:       "smtp.gmail.com:465",
		FromEmail:     "arash.cordi@gmail.com",
		WebServer:     "www.z42.com",
		ApiServer:     "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "../../templates/*.tmpl",
		Auth: Auth{
			Username: "arash.cordi@gmail.com",
			Password: "tsyuzcvdhshhremx",
		},
	})
	Expect(err).To(BeNil())
	err = m.SendEMailVerification("arash", "arash.cordi@gmail.com", "test")
	Expect(err).To(BeNil())
}

func TestSendWithAuthAWS(t *testing.T) {
	RegisterTestingT(t)
	m, err := NewSMTP(&Config{
		Address:       "email-smtp.eu-west-3.amazonaws.com:465",
		FromEmail:     "arash.cordi@gmail.com",
		WebServer:     "www.z42.com",
		ApiServer:     "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "../../templates/*.tmpl",
		Auth: Auth{
			Username: "AKIAXSERLOKQN6CFWN5Y",
			Password: "BOb0SYEGZ0w444OtHZ8ReQvqhcGbOm+Mql6XU1ImoASn",
		},
	})
	Expect(err).To(BeNil())
	err = m.SendEMailVerification("arash", "arash.cordi@gmail.com", "test")
	Expect(err).To(BeNil())
}
