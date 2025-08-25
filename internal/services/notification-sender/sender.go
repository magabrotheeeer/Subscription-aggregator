package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type SenderService struct {
	cfg *config.Config
	log *slog.Logger
}

// NewSenderService создает новый экземпляр SenderService.
func NewSenderService(cfg *config.Config, log *slog.Logger) *SenderService {
	return &SenderService{
		cfg: cfg,
		log: log,
	}
}

func (s *SenderService) SendInfoExpiringSubscription(body []byte) error {
	var message models.EntryInfo
	if err := json.Unmarshal(body, &message); err != nil {
		return fmt.Errorf("error unmarshalling message: %w", err)
	}

	to := []string{message.Email}

	subject := "Уведомление о скором окончании подписки"
	bodyText := fmt.Sprintf("Здравствуйте, %s!\n\nВаша подписка на сервис %s заканчивается завтра.\n\nПожалуйста, продлите её заранее.",
		message.Username, message.ServiceName)

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", s.cfg.SMTPUser),
		fmt.Sprintf("To: %s", strings.Join(to, ";")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=\"UTF-8\"",
		"",
		bodyText,
	}, "\r\n")

	addr := s.cfg.SMTPHost + ":" + s.cfg.SMTPPort

	// 1. Устанавливаем обычное TCP соединение
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		s.log.Error("failed to dial SMTP server", sl.Err(err))
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}

	// 2. Создаем SMTP клиента поверх TCP
	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		s.log.Error("failed to create SMTP client", sl.Err(err))
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// 3. Переключаемся на TLS с помощью STARTTLS
	tlsConfig := &tls.Config{
		ServerName: s.cfg.SMTPHost,
	}
	ok, _ := client.Extension("STARTTLS")
	if !ok {
		s.log.Error("SMTP server does not support STARTTLS")
		return fmt.Errorf("smtp server does not support STARTTLS")
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		s.log.Error("failed to start TLS", sl.Err(err))
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// 4. Аутентификация
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	if err = client.Auth(auth); err != nil {
		s.log.Error("smtp auth failed", sl.Err(err))
		return fmt.Errorf("smtp auth failed: %w", err)
	}

	// 5. Устанавливаем "от кого"
	if err = client.Mail(s.cfg.SMTPUser); err != nil {
		s.log.Error("failed to set mail sender", sl.Err(err))
		return fmt.Errorf("failed to set mail sender: %w", err)
	}

	// 6. Устанавливаем получателей
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			s.log.Error("failed to set recipient", sl.Err(err))
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	// 7. Записываем тело письма
	wc, err := client.Data()
	if err != nil {
		s.log.Error("failed to get write closer", sl.Err(err))
		return fmt.Errorf("failed to get write closer: %w", err)
	}
	_, err = wc.Write([]byte(msg))
	if err != nil {
		s.log.Error("failed to write message", sl.Err(err))
		return fmt.Errorf("failed to write message: %w", err)
	}
	err = wc.Close()
	if err != nil {
		s.log.Error("failed to close write closer", sl.Err(err))
		return fmt.Errorf("failed to close write closer: %w", err)
	}

	// 8. Завершаем работу клиента
	if err = client.Quit(); err != nil {
		s.log.Error("failed to quit SMTP client", sl.Err(err))
		return fmt.Errorf("failed to quit SMTP client: %w", err)
	}

	s.log.Info("email sent successfully", "to", to)
	return nil
}
