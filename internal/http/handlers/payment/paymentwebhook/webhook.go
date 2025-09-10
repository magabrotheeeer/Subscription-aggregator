// Package paymentwebhook обрабатывает webhook-запросы от платежных провайдеров.
package paymentwebhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Service определяет интерфейс для операций с платежами.
type Service interface {
	SavePayment(ctx context.Context, payload *Payload) (int, error)
	UpdateStatusActiveForSubscription(ctx context.Context, userUID string) error
	UpdateStatusCancelForSubscription(ctx context.Context, userUID string) error
}

// SenderService определяет интерфейс для отправки уведомлений.
type SenderService interface {
	SendInfoFailurePayment(payload *Payload) error
	SendInfoSuccessPayment(payload *Payload) error
}

// Handler обрабатывает webhook-запросы от платежных провайдеров.
type Handler struct {
	log            *slog.Logger // Логгер для записи информации и ошибок
	paymentService Service
	senderService  SenderService
	webhookSecret  string // Секрет для проверки подписи
}

// New создает новый экземпляр Handler.
func New(log *slog.Logger, paymentService Service, senderService SenderService, secret string) *Handler {
	return &Handler{
		log:            log,
		paymentService: paymentService,
		senderService:  senderService,
		webhookSecret:  secret,
	}
}

const (
	// PaymentSucceeded статус успешного платежа.
	PaymentSucceeded = "payment.succeeded"
	// PaymentCanceled статус успешного платежа.
	PaymentCanceled = "payment.canceled"
)

// Payload представляет структуру данных webhook-запроса от платежного провайдера.
type Payload struct {
	Event  string `json:"event"`
	Object struct {
		ID     string `json:"id"`     // payment ID
		Status string `json:"status"` // статус платежа
		Amount struct {
			Value    string `json:"value"`    // сумма в строке, например "100.00"
			Currency string `json:"currency"` // валюта
		} `json:"amount"`
		PaymentMethod struct {
			ID string `json:"id"` // платёжный метод (card_id)
		} `json:"payment_method"`
		Metadata map[string]string `json:"metadata"` // для user_uid, subscription_id и др.
	} `json:"object"`
}

func (h *Handler) verifySignature(secret string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// ServeHTTP godoc
// @Summary Webhook для обработки уведомлений от YooKassa
// @Description Обрабатывает уведомления о статусе платежей от платежного провайдера YooKassa
// @Tags Payments
// @Accept  json
// @Produce  json
// @Param X-Api-Signature header string true "Подпись webhook для проверки подлинности"
// @Param payload body Payload true "Данные уведомления от YooKassa"
// @Success 200 "Webhook обработан успешно"
// @Failure 400 "Некорректные данные webhook"
// @Failure 401 "Неверная подпись webhook"
// @Failure 500 "Ошибка сервера при обработке webhook"
// @Router /payments/webhook [post]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.payment.webhook"
	log := h.log.With(slog.String("op", op))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read webhook body", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Error("failed to close request body", "error", err)
		}
	}()

	signature := r.Header.Get("X-Api-Signature")
	if signature == "" || !h.verifySignature(h.webhookSecret, body, signature) {
		log.Error("invalid or missing webhook signature")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error("failed to unmarshal webhook payload", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := h.paymentService.SavePayment(r.Context(), &payload); err != nil {
		log.Error("failed to process success payment", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch strings.ToLower(payload.Event) {
	case PaymentSucceeded:
		err := h.senderService.SendInfoSuccessPayment(&payload)
		if err != nil {
			log.Error("failed to send info about success payment", sl.Err(err))
		}
		err = h.paymentService.UpdateStatusActiveForSubscription(r.Context(), payload.Object.Metadata["user_uid"])
		if err != nil {
			log.Error("failed to update status", sl.Err(err))
		}
	case PaymentCanceled:
		err := h.senderService.SendInfoFailurePayment(&payload)
		if err != nil {
			log.Error("failed to send info about failure payment", sl.Err(err))
		}
		err = h.paymentService.UpdateStatusCancelForSubscription(r.Context(), payload.Object.Metadata["user_uid"])
		if err != nil {
			log.Error("failed to update status", sl.Err(err))
		}
	default:
		log.Info("ignored webhook event", slog.String("event", payload.Event))
	}

	log.Info("webhook processed successfully", slog.String("event", payload.Event), slog.String("payment_id", payload.Object.ID))
	w.WriteHeader(http.StatusOK)
}
