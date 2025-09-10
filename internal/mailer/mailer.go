package mailer

import (
	"fmt"
	"log"
	"net/smtp"
)

type Mail struct {
	To      string
	Subject string
	Body    string
}

type Sender interface {
	Send(m Mail) error
}

type SMTPSender struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func (s *SMTPSender) Send(m Mail) error {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.User, s.Pass, s.Host)
	msg := []byte("To: " + m.To + "\r\nSubject: " + m.Subject + "\r\n\r\n" + m.Body)
	return smtp.SendMail(addr, auth, s.From, []string{m.To}, msg)
}

type LogSender struct{}

func (LogSender) Send(m Mail) error {
	log.Printf("MAIL to=%s subject=%s body=%s", m.To, m.Subject, m.Body)
	return nil
}
