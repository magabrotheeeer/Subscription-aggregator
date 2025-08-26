package smtp

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

type Transport struct {
	cfg *config.Config
	log *slog.Logger
}

func NewTransport(cfg *config.Config, log *slog.Logger) *Transport {
	return &Transport{cfg: cfg, log: log}
}

func (t *Transport) Connect() (*smtp.Client, error) {
	addr := t.cfg.SMTPHost + ":" + t.cfg.SMTPPort

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.log.Error("failed to dial SMTP server", sl.Err(err))
		return nil, fmt.Errorf("failed to dial SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, t.cfg.SMTPHost)
	if err != nil {
		t.log.Error("failed to create SMTP client", sl.Err(err))
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	tlsConfig := &tls.Config{ServerName: t.cfg.SMTPHost}
	ok, _ := client.Extension("STARTTLS")
	if !ok {
		t.log.Error("SMTP server does not support STARTTLS")
		_ = client.Close()
		return nil, fmt.Errorf("smtp server does not support STARTTLS")
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		t.log.Error("failed to start TLS", sl.Err(err))
		client.Close()
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}

	auth := smtp.PlainAuth("", t.cfg.SMTPUser, t.cfg.SMTPPass, t.cfg.SMTPHost)
	if err = client.Auth(auth); err != nil {
		t.log.Error("smtp auth failed", sl.Err(err))
		client.Close()
		return nil, fmt.Errorf("smtp auth failed: %w", err)
	}

	return client, nil
}

func (t *Transport) GetSMTPUser() string {
	return t.cfg.SMTPUser
}
