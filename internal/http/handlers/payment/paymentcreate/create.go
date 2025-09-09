package paymentcreate

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/paymentprovider"
)

type CreatePaymentMethodRequestApp struct {
	PaymentMethodToken string `json:"payment_method_token"`
}

type ProviderClient interface {
	CreatePayment(reqParams paymentprovider.CreatePaymentRequest) (*paymentprovider.CreatePaymentResponse, error)
}

type PaymentService interface {
	GetOrCreatePaymentToken(context context.Context, userUID string, token string) (int, error)
	GetActiveSubscriptionIDByUserUID(ctx context.Context, userUID string) (string, error)
}

type Handler struct {
	log            *slog.Logger   // Логгер для записи информации и ошибок
	providerClient ProviderClient // Клиeнт для работы с провайдером
	paymentService PaymentService
	validate       *validator.Validate // Валидатор структуры входящих данных
}

func New(log *slog.Logger, providerClient ProviderClient, ps PaymentService) *Handler {
	return &Handler{
		log:            log,
		providerClient: providerClient,
		paymentService: ps,
		validate:       validator.New(),
	}
}

// ServeHTTP возвращает клиенту confirmation_token, полученный от ЮКАССа
// Точка приема платежа от frontend
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.payment.create"
	log := h.log.With(slog.String("op", op))

	var req CreatePaymentMethodRequestApp
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		w.WriteHeader(http.StatusUnprocessableEntity)
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}

	userUID, ok := r.Context().Value(middlewarectx.UserUID).(string)
	if !ok || userUID == "" {
		log.Error("user UID not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}
	subscriptionID, err := h.paymentService.GetActiveSubscriptionIDByUserUID(r.Context(), userUID)
	if err != nil {
		log.Error("failed to get active subscription", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("internal error"))
		return
	}
	_, err = h.paymentService.GetOrCreatePaymentToken(r.Context(), userUID, req.PaymentMethodToken)
	if err != nil {
		log.Error("failed to create or read payment token", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("internal error"))
		return
	}

	paymentReq := paymentprovider.CreatePaymentRequest{
		PaymentToken: req.PaymentMethodToken,
		Amount: paymentprovider.Amount{
			Value:    "200.00",
			Currency: "RUB",
		},
		Metadata: map[string]string{
			"user_uid":        userUID,
			"subscription_id": subscriptionID,
		},
	}

	paymentResp, err := h.providerClient.CreatePayment(paymentReq)
	if err != nil {
		log.Error("failed to create payment method from provider", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("payment provider error"))
		return
	}

	log.Info("success to create payment method", slog.Any("payment-resp", paymentResp))
	render.JSON(w, r, paymentResp)
}
