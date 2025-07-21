package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/eldeeishere/cautious-octo-dollop/internal/auth"
	"github.com/eldeeishere/cautious-octo-dollop/internal/database"
	"github.com/google/uuid"
)

type adminData struct {
	Count int
}

type apiCreateUserReturn struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type apiConfig struct {
	TotalReq      atomic.Int32
	adminTemplate *template.Template
	database      *database.Queries
	tokenSecret   string
	apiKey        string
}

type ChirpRequest struct {
	Body string `json:"body"`
}

type ChirpResponse struct {
	Valid       bool   `json:"valid"`
	CleanedBody string `json:"cleaned_body,omitempty"`
	Error       string `json:"error,omitempty"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.TotalReq.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareMetricsGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	count := int(cfg.TotalReq.Load())
	data := adminData{Count: count}

	err := cfg.adminTemplate.Execute(w, data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

func (cfg *apiConfig) middlewareMetricsReset(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("PLATFORM") != "dev" {
		http.Error(w, "Resetting metrics is not allowed on Heroku", http.StatusForbidden)
		return
	}
	cfg.database.DeleteUser(r.Context())
	cfg.TotalReq.Store(0)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) middlewareNoBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 {
			respondWithError(w, http.StatusBadRequest, "Request body not allowed", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) apiCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}
	hashPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}
	user, err := cfg.database.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashPass,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create user", err)
		return
	}
	respondWithJSON(w, http.StatusCreated, apiCreateUserReturn{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	})

}

func (cfg *apiConfig) handlerChirpsLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type returnVals struct {
		Id           uuid.UUID `json:"id"`
		CreatedAt    string    `json:"created_at"`
		UpdatedAt    string    `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token,omitempty"`
		RefreshToken string    `json:"refresh_token,omitempty"`
		IsChirpyRed  bool      `json:"is_chirpy_red,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	if params.Email == "" || params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Email and password are required", nil)
		return
	}
	user, err := cfg.database.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user by email", err)
		return
	}
	if user.ID == uuid.Nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}
	sda := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if sda != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}

	refreshToken, jwtToken, err := cfg.CreateTokenAndRefreshToken(r.Context(), user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create tokens", err)
		return
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Email:        user.Email,
		Token:        jwtToken,
		RefreshToken: refreshToken,
		IsChirpyRed:  user.IsChirpyRed,
	})

}

func (cfg *apiConfig) CreateTokenAndRefreshToken(ctx context.Context, user database.User) (refreshToken, jwtToken string, err error) {
	jwtToken, err = auth.MakeJWT(user.ID, os.Getenv("SIG_SECRET"), time.Hour)
	if err != nil {
		return "", "", fmt.Errorf("couldn't create JWT token: %w", err)
	}

	refreshToken, err = auth.MakeRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("couldn't create refresh token: %w", err)
	}

	expiresAt := time.Now().Add(60 * 24 * time.Hour) // 60 days
	_, err = cfg.database.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", "", fmt.Errorf("couldn't save refresh token to database: %w", err)
	}

	return refreshToken, jwtToken, nil
}

func (cfg *apiConfig) handlerRefreshTokens(w http.ResponseWriter, r *http.Request) {
	type respondVals struct {
		Token string `json:"token"`
	}
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}
	auths, err := cfg.database.GetUserFromRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user from refresh token", err)
		return
	}
	jwtToekn, err := auth.MakeJWT(auths.ID, os.Getenv("SIG_SECRET"), time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create JWT token", err)
		return
	}
	respondWithJSON(w, http.StatusOK, respondVals{
		Token: jwtToekn,
	})

}

func (cfg *apiConfig) handlerRevokRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}
	err = cfg.database.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke refresh token", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type respondVals struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	if params.Email == "" && params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Email and password is required", nil)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}
	auths, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}
	if auths == uuid.Nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid user ID in token", nil)
		return
	}
	hashedPass, _ := auth.HashPassword(params.Password)
	_, err = cfg.database.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             auths,
		Email:          params.Email,
		HashedPassword: hashedPass,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update user", err)
		return
	}
	respondWithJSON(w, http.StatusOK, respondVals{
		Email: params.Email,
	})

}

func (cfg *apiConfig) handlerDeleteChirps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}
	auths, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}
	if auths == uuid.Nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid user ID in token", nil)
		return
	}

	idStrg := r.PathValue("chirpID")
	if idStrg == "" {
		respondWithError(w, http.StatusBadRequest, "Chirp ID is required", nil)
		return
	}
	message, err := cfg.database.GetMessageByID(r.Context(), uuid.MustParse(idStrg))
	if message.UserID != auths {
		respondWithError(w, http.StatusForbidden, "You are not allowed to delete this chirp", nil)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}
	cfg.database.DeleteChirpsByID(r.Context(), database.DeleteChirpsByIDParams{
		ID:     uuid.MustParse(idStrg),
		UserID: auths,
	})
	w.WriteHeader(http.StatusNoContent)
}

type Event struct {
	Event string    `json:"event"`
	Data  EventData `json:"data"`
}

type EventData struct {
	UserID string `json:"user_id"`
}

func (cfg *apiConfig) handlerAddSubscription(w http.ResponseWriter, r *http.Request) {
	event := Event{}
	apiKey, err := auth.GetApiKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing API key", err)
		return
	}
	if apiKey != cfg.apiKey {
		respondWithError(w, http.StatusUnauthorized, "Invalid API key", nil)
		return
	}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&event)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	if event.Event != "user.upgraded" {
		respondWithJSON(w, http.StatusNoContent, nil)
		return
	}
	err = cfg.database.AddUserChirpyRed(r.Context(), uuid.MustParse(event.Data.UserID))
	if err != nil {
		respondWithJSON(w, http.StatusNotFound, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	respondWithJSON(w, http.StatusNoContent, nil)

}
