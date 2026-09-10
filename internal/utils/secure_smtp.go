package utils

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	"kori/internal/models"
	"net"
	"net/mail"
	"net/smtp"
	"time"
)

// sendSecureSMTP requires TLS before authentication and never skips certificate checks.
func sendSecureSMTP(message *gomail.Message, email *models.Email) error {
	config := email.SMTPConfig
	address := net.JoinHostPort(config.Host, fmt.Sprint(config.Port))
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: config.Host}
	dialer := &net.Dialer{Timeout: 20 * time.Second}
	var conn net.Conn
	var err error
	if config.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(45 * time.Second)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if config.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server must support STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(email.From); err != nil {
		return err
	}
	for _, header := range []string{"To", "Cc", "Bcc"} {
		for _, recipient := range message.GetHeader(header) {
			addr, err := mail.ParseAddress(recipient)
			if err != nil {
				return err
			}
			if err := client.Rcpt(addr.Address); err != nil {
				return err
			}
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = message.WriteTo(writer); err != nil {
		return fmt.Errorf("delivery outcome unknown: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("delivery outcome unknown: %w", err)
	}
	// DATA was acknowledged; a QUIT failure must not cause a duplicate delivery.
	_ = client.Quit()
	return nil
}
