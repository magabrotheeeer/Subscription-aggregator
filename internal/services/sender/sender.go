// Package services предоставляет бизнес-логику приложения.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/smtp"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// SubscriptionRepository определяет интерфейс для работы с подписками в репозитории.
type SubscriptionRepository interface {
	GetUser(ctx context.Context, userUID string) (*models.User, error)
}

// SenderService предоставляет сервис для отправки уведомлений.
type SenderService struct {
	repo      SubscriptionRepository
	transport smtp.TransportInterface
	log       *slog.Logger
}

// NewSenderService создает новый экземпляр SenderService.
func NewSenderService(repo SubscriptionRepository, log *slog.Logger, transport smtp.TransportInterface) *SenderService {
	return &SenderService{
		repo:      repo,
		transport: transport,
		log:       log,
	}
}

// SendInfoExpiringSubscription отправляет уведомление об истекающей подписке.
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

// SendInfoExpiringTrialPeriodSubscription отправляет уведомление об истекающем пробном периоде.
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
		`, message.Username, "ссылка_на_оплату")

	return s.sendEmail(to, subject, bodyText)
}

// SendInfoSuccessPayment отправляет уведомление об успешном платеже.
func (s *SenderService) SendInfoSuccessPayment(payload *paymentwebhook.Payload) error {
	user, err := s.repo.GetUser(context.Background(), payload.Object.Metadata["user_uid"])
	if err != nil {
		s.log.Error("Failed to get username", "error", sl.Err(err))
		return fmt.Errorf("failed to get username: %w", err)
	}
	to := []string{user.Email}
	subject := "Уведомление об успешном списании денежных средств на Subscription-aggregator"
	bodyText := fmt.Sprintf(`Здравствуйте, %s!
			С вашего счёта успешно списана сумма за подписку на сервис Subscription-aggregator.
			Спасибо за использование нашего сервиса!
		`, user.Username)
	return s.sendEmail(to, subject, bodyText)
}

// SendInfoFailurePayment отправляет уведомление о неудачном платеже.
func (s *SenderService) SendInfoFailurePayment(payload *paymentwebhook.Payload) error {
	user, err := s.repo.GetUser(context.Background(), payload.Object.Metadata["user_uid"])
	if err != nil {
		s.log.Error("Failed to get username", "error", sl.Err(err))
		return fmt.Errorf("failed to get username: %w", err)
	}
	to := []string{user.Email}
	subject := "Уведомление о неуспешном списании денежных средств на Subscription-aggregator"
	bodyText := fmt.Sprintf(`Здравствуйте, %s!
			К сожалению, с вашего счёта не удалось списать оплату за подписку на сервис Subscription-aggregator.
			Для повторной оплаты перейдите по ссылке: %s
		`, user.Username, "ссылка_на_оплату")
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
	defer func() {
		if err := client.Close(); err != nil {
			s.log.Error("failed to close SMTP client", "error", sl.Err(err))
		}
	}()

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
