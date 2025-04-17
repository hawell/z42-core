package mailer

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/smtp"
)

type Mailer interface {
	SendEMailVerification(toName string, toEmail string, code string) error
	SendPasswordReset(toName string, toEmail string, code string) error
}

type Mock struct {
	SendEMailVerificationFunc func(toName string, toEmail string, code string) error
	SendPasswordResetFunc     func(toName string, toEmail string, code string) error
}

func (m *Mock) SendEMailVerification(toName string, toEmail string, code string) error {
	return m.SendEMailVerificationFunc(toName, toEmail, code)
}

func (m *Mock) SendPasswordReset(toName string, toEmail string, code string) error {
	return m.SendPasswordResetFunc(toName, toEmail, code)
}

type SMTP struct {
	config *Config
	tmpl   *template.Template
}

func NewSMTP(config *Config) (Mailer, error) {
	tmpl, err := template.ParseGlob(config.HtmlTemplates)
	if err != nil {
		return nil, err
	}
	m := &SMTP{
		config: config,
		tmpl:   tmpl,
	}
	return m, nil
}

func (m *SMTP) SendEMailVerification(toName string, toEmail string, code string) error {
	var b bytes.Buffer
	err := m.tmpl.ExecuteTemplate(
		&b,
		"verification-email.tmpl",
		struct {
			Server string
			Code   string
		}{
			Server: m.config.ApiServer,
			Code:   code,
		})
	if err != nil {
		return err
	}
	return m.send(toName, toEmail, "email verification", b.String())
}

func (m *SMTP) SendPasswordReset(toName string, toEmail string, code string) error {
	var b bytes.Buffer
	err := m.tmpl.ExecuteTemplate(
		&b,
		"password-reset-email.tmpl",
		struct {
			Server string
			Code   string
		}{
			Server: m.config.WebServer,
			Code:   code,
		})
	if err != nil {
		return err
	}
	return m.send(toName, toEmail, "password reset", b.String())
}

func (m *SMTP) send(toName string, toEmail string, subject string, body string) error {
	var (
		c   *smtp.Client
		err error
	)
	host, _, err := net.SplitHostPort(m.config.Address)
	if err != nil {
		return err
	}
	if m.config.Auth.Username != "" {
		conn, err := net.Dial("tcp", m.config.Address)
		if err != nil {
			return err
		}
		c, err = smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		if err = c.StartTLS(tlsConfig); err != nil {
			return err
		}
		err = c.Auth(smtp.PlainAuth("", m.config.Auth.Username, m.config.Auth.Password, host))
		if err != nil {
			return err
		}
	} else {
		c, err = smtp.Dial(m.config.Address)
		if err != nil {
			return err
		}
	}
	defer c.Close()

	to := mail.Address{Name: toName, Address: toEmail}
	header := make(map[string]string)
	header["To"] = to.String()
	header["From"] = m.config.FromEmail
	header["Subject"] = subject
	header["Content-Type"] = `text/html; charset="UTF-8"`
	msg := ""
	for k, v := range header {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body
	bMsg := []byte(msg)
	if err = c.Mail(m.config.FromEmail); err != nil {
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
