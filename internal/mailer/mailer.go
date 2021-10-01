package mailer

import (
	"fmt"
	"net/mail"
	"net/smtp"
)

type Mailer interface {
	Send(toName string, toEmail string, subject string, body string) error
}

type Mock struct {
	SendFunc func(toName string, toEmail string, subject string, body string) error
}

func (m *Mock) Send(toName string, toEmail string, subject string, body string) error {
	return m.SendFunc(toName, toEmail, subject, body)
}

type SMTP struct {
	config *Config
}

func NewSMTP(config *Config) Mailer {
	return &SMTP{config: config}
}

func (m *SMTP) Send(toName string, toEmail string, subject string, body string) error {
	c, err := smtp.Dial(m.config.Address)
	if err != nil {
		return err
	}
	defer c.Close()
	from := mail.Address{Name: m.config.FromName, Address: m.config.FromEmail}
	fromHeader := from.String()

	to := mail.Address{Name: toName, Address: toEmail}
	header := make(map[string]string)
	header["To"] = to.String()
	header["From"] = fromHeader
	header["Subject"] = subject
	header["Content-Type"] = `text/html; charset="UTF-8"`
	msg := ""
	for k, v := range header {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body
	bMsg := []byte(msg)
	if err = c.Mail(fromHeader); err != nil {
		return err
	}
	err = c.Rcpt(toEmail)
	if err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(bMsg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}