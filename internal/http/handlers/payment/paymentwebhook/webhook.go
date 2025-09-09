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

type PaymentService interface {
	SavePayment(ctx context.Context, payload *Payload) (int, error)
	UpdateStatusActiveForSubscription(ctx context.Context, userUID string) error
	UpdateStatusCancelForSubscription(ctx context.Context, userUID string) error
}

type SenderService interface {
	SendInfoFailurePayment(payload *Payload) error
	SendInfoSuccessPayment(payload *Payload) error
}

type Handler struct {
	log            *slog.Logger // Логгер для записи информации и ошибок
	paymentService PaymentService
	senderService  SenderService
	webhookSecret  string // Секрет для проверки подписи
}

func New(log *slog.Logger, paymentService PaymentService, senderService SenderService, secret string) *Handler {
	return &Handler{
		log:            log,
		paymentService: paymentService,
		senderService:  senderService,
		webhookSecret:  secret,
	}
}

const (
	PaymentSucceeded = "payment.succeeded"
	// PaymentWaitingForCapture = "payment.waiting_for_capture"
	PaymentCanceled = "payment.canceled"
	// PaymentRefunded          = "payment.refunded"
	// PaymentWaitingForAction  = "payment.waiting_for_action"
)

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
