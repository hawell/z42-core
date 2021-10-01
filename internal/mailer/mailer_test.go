package mailer

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestSend(t *testing.T) {
	RegisterTestingT(t)
	m := NewSMTP(&Config{
		Address:   "127.0.0.1:25",
		FromName:  "z42",
		FromEmail: "noreply@zone-42.com",
	})
	err := m.Send("arash", "arash@legioner", "test", "test from z42")
	Expect(err).To(BeNil())
}