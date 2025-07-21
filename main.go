package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/eldeeishere/cautious-octo-dollop/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func NewApiConfig(db *sql.DB, secret, apikey string) *apiConfig {
	tmpl, err := template.ParseFiles("./admin/admin.html")
	if err != nil {
		log.Fatal("Error loading admin template:", err)
	}
	dbQueries := database.New(db)
	return &apiConfig{
		adminTemplate: tmpl,
		database:      dbQueries,
		tokenSecret:   secret,
		apiKey:        apikey,
	}
}

func main() {
	mux := http.NewServeMux()

	godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("Error pinging the database:", err)
	}
	apiCfg := NewApiConfig(db, os.Getenv("SIG_SECRET"), os.Getenv("POLKA_KEY"))

	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.HandleFunc("GET /api/healthz", endpointHealt)
	mux.HandleFunc("GET /admin/metrics", apiCfg.middlewareMetricsGet)
	mux.HandleFunc("POST /admin/reset", apiCfg.middlewareMetricsReset)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirpsValidate)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerChirpsGetAll)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerChirpsGetByID)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirps)
	mux.HandleFunc("POST /api/users", apiCfg.apiCreateUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	mux.HandleFunc("POST /api/login", apiCfg.handlerChirpsLogin)
	mux.Handle("POST /api/refresh", apiCfg.middlewareNoBody(http.HandlerFunc(apiCfg.handlerRefreshTokens)))
	mux.Handle("POST /api/revoke", apiCfg.middlewareNoBody(http.HandlerFunc(apiCfg.handlerRevokRefreshToken)))
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerAddSubscription)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}

func endpointHealt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
