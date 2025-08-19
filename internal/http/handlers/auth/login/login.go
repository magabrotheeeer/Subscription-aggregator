package login

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"

	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/client"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Request — структура входных данных для авторизации
type Request struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type Handler struct {
	log        *slog.Logger
	authClient *client.AuthClient
	validate   *validator.Validate
}

func New(log *slog.Logger, authClient *client.AuthClient) *Handler {
	return &Handler{
		log:        log,
		authClient: authClient,
		validate:   validator.New(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.auth.login"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		render.JSON(w, r, response.Error("invalid request body"))
		return
	}
	log.Info("request body decoded", slog.Any("request", req))

	if err := h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}
	log.Info("all fields are validated")

	grpcResp, err := h.authClient.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		log.Error("login failed", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("invalid credentials"))
		return
	}

	log.Info("login success", slog.Any("username", req.Username))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"token":         grpcResp.Token,
		"refresh_token": grpcResp.RefreshToken,
		"role":          grpcResp.Role,
		"username":      req.Username,
	}))
}
