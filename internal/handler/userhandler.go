package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/TMWF/gopher-mart/internal/util"
	"github.com/TMWF/gopher-mart/internal/util/validation"
	"github.com/go-playground/validator"
)

type userHandler struct {
	userService service.UserService
	logger      *slog.Logger
	jwtBuilder  util.UserJWTBuilder
	validator   *validator.Validate
}

func NewUserHandler(
	us service.UserService,
	logger *slog.Logger,
	jwtBuilder util.UserJWTBuilder,
	validator *validator.Validate,
) *userHandler {

	return &userHandler{
		userService: us,
		logger:      logger.With(slog.String("op", "handler.UserHandler")),
		jwtBuilder:  jwtBuilder,
		validator:   validator,
	}
}

// RegisterUser handles the HTTP request for new user registration.
//
// The handler follows a standard web-service workflow:
//  1. Validates that the HTTP method is POST.
//  2. Decodes the JSON request body into a UserRegisterRequestModel.
//  3. Performs structural and logic validation of the input data.
//  4. Invokes the service layer to create the user and hash their password.
//  5. Generates a JWT (JSON Web Token) for the newly created user.
//  6. Sets an "HttpOnly" cookie containing the JWT to ensure secure authentication.
//
// HTTP Response Codes:
//   - 200 OK: User successfully registered and authenticated (JWT cookie set).
//   - 400 Bad Request: Invalid JSON format or failed field validation.
//   - 405 Method Not Allowed: Request method is not POST.
//   - 409 Conflict: A user with the provided login already exists.
//   - 500 Internal Server Error: Unexpected failure during user creation or token generation.
func (h *userHandler) RegisterUser(w http.ResponseWriter, req *http.Request) {
	log := h.logger.With(slog.String("op", "handler.RegisterUser"))

	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody model.UserRegisterRequestModel
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqBody); err != nil {
		log.Error("Error occured while decoding request body", logger.Err(err))
		http.Error(w, "Error occured while decoding request body", http.StatusBadRequest)
		return
	}

	if !validation.IsValidRequest(reqBody, h.validator, log) {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := h.userService.CreateUser(req.Context(), &reqBody)

	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			log.Error(err.Error())
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jwtToken, err := h.jwtBuilder.BuildJWTString(*userID)
	if err != nil {
		log.Error(err.Error())
		http.Error(w, "Unexpected error occured while trying to get jwt", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{Name: string(util.UserID), Value: jwtToken, HttpOnly: true, MaxAge: 3600 * 24}
	http.SetCookie(w, &cookie)
	log.Debug("Successfully set jwt cookie")

	w.WriteHeader(http.StatusOK)
}

// LoginUser handles the user authentication process via an HTTP POST request.
//
// The method performs the following steps:
//  1. Verifies that the HTTP method is POST.
//  2. Decodes the JSON request body into [model.UserLoginRequestModel].
//  3. Validates the input fields using the configured validator.
//  4. Authenticates the user credentials through the User Service.
//  5. Generates a JWT token upon successful authentication.
//  6. Sets an "HttpOnly" cookie containing the JWT with a 24-hour expiration.
//
// HTTP Responses:
//   - 200 OK: Login successful; JWT cookie is set.
//   - 400 Bad Request: Invalid JSON body or validation failed.
//   - 401 Unauthorized: Invalid login credentials (user does not exist or password mismatch).
//   - 405 Method Not Allowed: The request method is not POST.
//   - 500 Internal Server Error: Unexpected error during authentication or JWT generation.
func (h *userHandler) LoginUser(w http.ResponseWriter, req *http.Request) {
	log := h.logger.With(slog.String("op", "LoginUser"))

	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody model.UserLoginRequestModel
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqBody); err != nil {
		log.Error("Error occured while decoding request body", logger.Err(err))
		http.Error(w, "Error occured while decoding request body", http.StatusBadRequest)
		return
	}

	if !validation.IsValidRequest(reqBody, h.validator, log) {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := h.userService.LoginUser(req.Context(), &reqBody)
	if err != nil {
		if errors.Is(err, service.ErrUserDoesNotExist) {
			log.Error(err.Error())
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			return
		}

		log.Error(err.Error())
		http.Error(w, "Unexpected error occured while trying to login user", http.StatusInternalServerError)
		return
	}

	jwtToken, err := h.jwtBuilder.BuildJWTString(*userID)
	if err != nil {
		log.Error(err.Error())
		http.Error(w, "Unexpected error occured while trying to get jwt", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{Name: string(util.UserID), Value: jwtToken, HttpOnly: true, MaxAge: 3600 * 24}
	http.SetCookie(w, &cookie)
	log.Debug("Successfully set jwt cookie")

	w.WriteHeader(http.StatusOK)
}
