package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"wjfcm-go/internal/config"
)

func SendMail(cfg config.MailConfig, subject string, body string) error {
	return SendMailTo(cfg, cfg.To, subject, body)
}

func SendMailTo(cfg config.MailConfig, toAddress string, subject string, body string) error {
	if strings.ToLower(cfg.Driver) != "smtp" {
		return nil
	}
	if cfg.Host == "" || cfg.Port == "" || cfg.From == "" || toAddress == "" {
		return nil
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	from := cfg.From
	to := []string{toAddress}
	message := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from,
		toAddress,
		subject,
		body,
	))

	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	if isImplicitTLS(cfg) {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
		return sendSMTPMessage(client, from, to, message)
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if strings.EqualFold(cfg.Encryption, "tls") {
		if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	return sendSMTPMessage(client, from, to, message)
}

func isImplicitTLS(cfg config.MailConfig) bool {
	if strings.EqualFold(cfg.Encryption, "ssl") {
		return true
	}
	port, _ := strconv.Atoi(cfg.Port)
	return port == 465
}

func sendSMTPMessage(client *smtp.Client, from string, to []string, message []byte) error {
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, item := range to {
		if err := client.Rcpt(item); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
