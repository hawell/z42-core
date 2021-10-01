package mailer

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestSend(t *testing.T) {
	RegisterTestingT(t)
	m, err := NewSMTP(&Config{
		Address:       "127.0.0.1:25",
		FromName:      "z42",
		FromEmail:     "noreply@zone-42.com",
		WebServer:    "www.z42.com",
		ApiServer:    "api.z42.com",
		NameServer:   "ns.z42.com.",
		HtmlTemplates: "../../templates/*.tmpl",
	})
	Expect(err).To(BeNil())
	err = m.SendEMailVerification("arash", "arash@legioner", "test")
	Expect(err).To(BeNil())
}
