package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type SenderService struct {
	transport Transport
	log       *slog.Logger
}

type Transport interface {
	Connect() (*smtp.Client, error)
	GetSMTPUser() string
}

// NewSenderService создает новый экземпляр SenderService.
func NewSenderService(cfg *config.Config, log *slog.Logger, transport Transport) *SenderService {
	return &SenderService{
		transport: transport,
		log:       log,
	}
}

func (s *SenderService) SendInfoExpiringSubscription(body []byte) error {
	var message models.EntryInfo
	if err := json.Unmarshal(body, &message); err != nil {
		s.log.Error("Failed to unmarshal message body", "error", sl.Err(err))
		return fmt.Errorf("error unmarshalling message: %w", err)
	}

	to := []string{message.Email}
	subject := "Уведомление о скором окончании подписки"
	bodyText := fmt.Sprintf("Здравствуйте, %s!\n\nВаша подписка на сервис %s заканчивается завтра.\n\nПожалуйста, продлите её заранее.",
		message.Username, message.ServiceName)

	return s.sendEmail(to, subject, bodyText)
}

func (s *SenderService) SendInfoExpiringTrialPeriodSubscription(body []byte) error {
	var message models.User
	if err := json.Unmarshal(body, &message); err != nil {
		s.log.Error("Failed to unmarshal message body", "error", sl.Err(err))
		return fmt.Errorf("error unmarshalling message: %w", err)
	}

	to := []string{message.Email}
	subject := "Уведомление о скором окончании пробного периода на Subscription-aggregator"
	bodyText := fmt.Sprintf(`Здравствуйте, %s!
			Ваша подписка на сервис Subscription-aggregator заканчивается сегодня.
			Если вы решите ее продлить, то для оплаты необходимо перейти по ссылке: %s.
			В противном случае сервис будет недоступен.
		`, message.Username, "ваша_ссылка_на_оплату")

	return s.sendEmail(to, subject, bodyText)
}

func (s *SenderService) sendEmail(to []string, subject, bodyText string) error {
	msg := strings.Join([]string{
		"From: " + s.transport.GetSMTPUser(),
		"To: " + strings.Join(to, ";"),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=\"UTF-8\"",
		"",
		bodyText,
	}, "\r\n")

	client, err := s.transport.Connect()
	if err != nil {
		s.log.Error("Failed to connect to SMTP server", "error", sl.Err(err))
		return err
	}
	defer client.Close()

	if err := client.Mail(s.transport.GetSMTPUser()); err != nil {
		s.log.Error("Failed to set MAIL FROM", "from", s.transport.GetSMTPUser(), "error", sl.Err(err))
		return err
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			s.log.Error("Failed to set RCPT TO", "recipient", addr, "error", sl.Err(err))
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		s.log.Error("Failed to get Data writer", "error", sl.Err(err))
		return err
	}

	_, err = wc.Write([]byte(msg))
	if err != nil {
		s.log.Error("Failed to write email body", "error", sl.Err(err))
		return err
	}

	if err = wc.Close(); err != nil {
		s.log.Error("Failed to close Data writer", "error", sl.Err(err))
		return err
	}

	if err = client.Quit(); err != nil {
		s.log.Error("Failed to quit SMTP client", "error", sl.Err(err))
		return err
	}

	s.log.Info("email sent successfully", "to", to)
	return nil
}
