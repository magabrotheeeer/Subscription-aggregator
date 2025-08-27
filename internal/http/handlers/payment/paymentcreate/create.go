package paymentcreate

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/paymentprovider"
)

type CreatePaymentMethodRequestApp struct {
	UserUID            string `json:"user_uid"`
	ProviderCustomerID string `json:"provider_customer_id"`
	PaymentMethodToken string `json:"payment_method_token"`
	CardBrand          string `json:"card_brand"`
	CardLastFour       string `json:"card_last_four"`
}

type ProviderService interface {
	CreatePaymentMethod(req paymentprovider.CreatePaymentMethodRequest) (*paymentprovider.CreatePaymentMethodResponse, error)
}

type AppService interface {
	//TODO: реализовать функции для работа с бизнес-слоем
	CreatePaymentMethod(paymentResponse paymentprovider.CreatePaymentMethodResponse)
}

type Handler struct {
	log             *slog.Logger    // Логгер для записи информации и ошибок
	providerService ProviderService // Сервис для работы с провайдером
	appService      AppService
	validate        *validator.Validate // Валидатор структуры входящих данных
}

func New(log *slog.Logger, providerService ProviderService, appService AppService) *Handler {
	return &Handler{
		log:             log,
		providerService: providerService,
		appService:      appService,
		validate:        validator.New(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.payment.create"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req paymentprovider.CreatePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("invalid request body"))
		return
	}
	log.Info("request body decoded", slog.Any("request", req))

	if err := h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		w.WriteHeader(http.StatusUnprocessableEntity)
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}
	log.Info("all fields are validated")

	useruid, ok := r.Context().Value(middlewarectx.UserUID).(string)
	if !ok || useruid == "" {
		log.Error("username not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}
	responseFromProvider, err := h.providerService.CreatePaymentMethod(req)
	if err != nil {
		log.Error("failed to get response from provider", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("failed to get response from provider"))
	}
	h.appService.CreatePaymentMethod(*responseFromProvider)

}
