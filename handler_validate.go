package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/eldeeishere/cautious-octo-dollop/internal/auth"
	"github.com/eldeeishere/cautious-octo-dollop/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerChirpsValidate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	type returnVals struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt string    `json:"created_at"`
		UpdatedAt string    `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
		Token     string    `json:"token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}
	auth, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}
	if auth == uuid.Nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid user ID in token", nil)
		return
	}
	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}
	messages, err := cfg.database.CreateMessage(r.Context(), database.CreateMessageParams{
		Body:   params.Body,
		UserID: auth,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create message", err)
		return
	}
	messages.Body = cleanProfanity(messages.Body)
	respondWithJSON(w, http.StatusCreated, &returnVals{
		Id:        messages.ID,
		CreatedAt: messages.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: messages.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Body:      messages.Body,
		UserID:    messages.UserID,
		Token:     token,
	})
}

func cleanProfanity(msg string) string {
	profaneWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	words := strings.Split(msg, " ")
	for i, word := range words {
		if profaneWords[strings.ToLower(word)] {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

type HttpQueriesOptions struct {
	author_id string
}

func (cfg *apiConfig) handlerChirpsGetAll(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt string    `json:"created_at"`
		UpdatedAt string    `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	author := r.URL.Query().Get("author_id")
	sorts := r.URL.Query().Get("sort")
	if sorts != "" && sorts != "asc" && sorts != "desc" {
		respondWithError(w, http.StatusBadRequest, "Invalid sort parameter", nil)
		return
	}

	messages, err := cfg.database.GetMessages(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get messages", err)
		return
	}
	var chirps []returnVals
	if sorts == "desc" {
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].CreatedAt.After(messages[j].CreatedAt)
		})
	} else if sorts == "asc" {
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		})
	}
	for _, msg := range messages {
		if author != "" && msg.UserID == uuid.MustParse(author) {
			chirps = append(chirps, returnVals{
				Id:        msg.ID,
				CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: msg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Body:      cleanProfanity(msg.Body),
				UserID:    msg.UserID,
			})
		} else if author == "" {
			chirps = append(chirps, returnVals{
				Id:        msg.ID,
				CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: msg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Body:      cleanProfanity(msg.Body),
				UserID:    msg.UserID,
			})
		}

	}
	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerChirpsGetByID(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt string    `json:"created_at"`
		UpdatedAt string    `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	idStrg := r.PathValue("chirpID")
	if idStrg == "" {
		respondWithError(w, http.StatusBadRequest, "Chirp ID is required", nil)
		return
	}

	chripts, err := cfg.database.GetMessageByID(r.Context(), uuid.MustParse(idStrg))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get message", err)
		return
	}
	respondWithJSON(w, http.StatusOK, &returnVals{
		Id:        chripts.ID,
		CreatedAt: chripts.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: chripts.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Body:      cleanProfanity(chripts.Body),
		UserID:    chripts.UserID,
	})
}
