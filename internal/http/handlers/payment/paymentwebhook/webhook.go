package paymentwebhook

import (
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

type Service interface {
	ProcessWebhookEvent(payload *Payload) error
}

type Handler struct {
	log           *slog.Logger // Логгер для записи информации и ошибок
	service       Service
	webhookSecret string // Секрет для проверки подписи
}

func New(log *slog.Logger, service Service, secret string) *Handler {
	return &Handler{
		log:           log,
		service:       service,
		webhookSecret: secret,
	}
}

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

// Проверка подписи webhook (X-Api-Signature)
func (h *Handler) verifySignature(secret string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	// Важно делать сравнение без уязвимостей по времени (хотя base64 строки)
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.payment.webhook"
	log := h.log.With(slog.String("op", op))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read webhook body", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Проверка подписи (в заголовке X-Api-Signature)
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

	// Обрабатываем только нужные события
	const (
		PaymentSucceeded         = "payment.succeeded"
		PaymentWaitingForCapture = "payment.waiting_for_capture"
		PaymentCanceled          = "payment.canceled"
		PaymentRefunded          = "payment.refunded"
		PaymentWaitingForAction  = "payment.waiting_for_action"
	)

	switch strings.ToLower(payload.Event) {
	case PaymentSucceeded,
		PaymentWaitingForCapture,
		PaymentCanceled,
		PaymentRefunded,
		PaymentWaitingForAction:
		if err := h.service.ProcessWebhookEvent(&payload); err != nil {
			log.Error("failed to process webhook event", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		log.Info("ignored webhook event", slog.String("event", payload.Event))
	}

	log.Info("webhook processed successfully", slog.String("event", payload.Event), slog.String("payment_id", payload.Object.ID))
	w.WriteHeader(http.StatusOK)
}
