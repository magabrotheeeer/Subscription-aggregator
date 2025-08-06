package login

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/user"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=6,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserGetter interface {
	GetUserByUsername(ctx context.Context, username string) (*user.User, error)
}

func New(ctx context.Context, log *slog.Logger, userGetter UserGetter, jwtMaker auth.JWTMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.login.New"
		var err error
		var loginRequest LoginRequest

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		err = render.DecodeJSON(r.Body, &loginRequest)
		if err != nil {
			log.Error("failed to decode request body", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to decode request"))

			return
		}
		log.Info("request body decoded", slog.Any("request", loginRequest))

		if err := validator.New().Struct(loginRequest); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})

			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		user, err := userGetter.GetUserByUsername(ctx, loginRequest.Username)
		if err != nil {
			log.Error("incorrect user or password", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("incorrect user or password"))

			return
		}

		err = auth.CompareHash(user.PasswordHash, loginRequest.Password)
		if err != nil {
			log.Error("incorrect user or password", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("incorrect user or password"))

			return
		}

		token, err := jwtMaker.GenerateToken(user.Username)
		if err != nil {
			log.Error("could not generate token", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("could not generate token"))

			return

		}
		log.Info("created token", "token", token)
		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"token": token,
		}))
	}
}
