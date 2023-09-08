package mailer

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestSend(t *testing.T) {
	RegisterTestingT(t)
	t.Skip("manual")
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
	t.Skip("manual")
	m, err := NewSMTP(&Config{
		Address:       "smtp.gmail.com:465",
		FromEmail:     "mail.zone42@gmail.com",
		WebServer:     "www.zone-42.com",
		ApiServer:     "api.zone-42.com",
		NameServer:    "ns.zone-42.com.",
		HtmlTemplates: "../../templates/*.tmpl",
		Auth: Auth{
			Username: "AAAA.AAAA@gmail.com",
			Password: "XXXXXXXX",
		},
	})
	Expect(err).To(BeNil())
	err = m.SendEMailVerification("arash", "arash.cordi@gmail.com", "test")
	Expect(err).To(BeNil())
}

func TestSendWithAuthAWS(t *testing.T) {
	RegisterTestingT(t)
	t.Skip("manual")
	m, err := NewSMTP(&Config{
		Address:       "email-smtp.eu-west-3.amazonaws.com:465",
		FromEmail:     "arash.cordi@gmail.com",
		WebServer:     "www.z42.com",
		ApiServer:     "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "../../templates/*.tmpl",
		Auth: Auth{
			Username: "AAAAAAAAAAAAAAAAAAAA",
			Password: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		},
	})
	Expect(err).To(BeNil())
	err = m.SendEMailVerification("arash", "arash.cordi@gmail.com", "test")
	Expect(err).To(BeNil())
}
