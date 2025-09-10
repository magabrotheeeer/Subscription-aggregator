package smtp

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/smtp"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Transport реализует SMTP транспорт для отправки писем.
type Transport struct {
	cfg *config.Config
	log *slog.Logger
}

// smtpClientWrapper обертка для *smtp.Client, реализующая интерфейс Client.
type smtpClientWrapper struct {
	client *smtp.Client
}

func (w *smtpClientWrapper) Mail(from string) error {
	return w.client.Mail(from)
}

func (w *smtpClientWrapper) Rcpt(to string) error {
	return w.client.Rcpt(to)
}

func (w *smtpClientWrapper) Data() (io.WriteCloser, error) {
	return w.client.Data()
}

func (w *smtpClientWrapper) Quit() error {
	return w.client.Quit()
}

func (w *smtpClientWrapper) Close() error {
	return w.client.Close()
}

// NewTransport создает новый экземпляр Transport.
func NewTransport(cfg *config.Config, log *slog.Logger) *Transport {
	return &Transport{cfg: cfg, log: log}
}

// Connect устанавливает соединение с SMTP сервером.
func (t *Transport) Connect() (Client, error) {
	addr := t.cfg.SMTPHost + ":" + t.cfg.SMTPPort

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.log.Error("failed to dial SMTP server", sl.Err(err))
		return nil, fmt.Errorf("failed to dial SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, t.cfg.SMTPHost)
	if err != nil {
		t.log.Error("failed to create SMTP client", sl.Err(err))
		if closeErr := conn.Close(); closeErr != nil {
			t.log.Error("failed to close connection", sl.Err(closeErr))
		}
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName: t.cfg.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}
	ok, _ := client.Extension("STARTTLS")
	if !ok {
		t.log.Error("SMTP server does not support STARTTLS")
		if closeErr := client.Close(); closeErr != nil {
			t.log.Error("failed to close client", sl.Err(closeErr))
		}
		return nil, fmt.Errorf("smtp server does not support STARTTLS")
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		t.log.Error("failed to start TLS", sl.Err(err))
		if closeErr := client.Close(); closeErr != nil {
			t.log.Error("failed to close client", sl.Err(closeErr))
		}
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}

	auth := smtp.PlainAuth("", t.cfg.SMTPUser, t.cfg.SMTPPass, t.cfg.SMTPHost)
	if err = client.Auth(auth); err != nil {
		t.log.Error("smtp auth failed", sl.Err(err))
		if closeErr := client.Close(); closeErr != nil {
			t.log.Error("failed to close client", sl.Err(closeErr))
		}
		return nil, fmt.Errorf("smtp auth failed: %w", err)
	}

	return &smtpClientWrapper{client: client}, nil
}

// GetSMTPUser возвращает имя пользователя SMTP.
func (t *Transport) GetSMTPUser() string {
	return t.cfg.SMTPUser
}
