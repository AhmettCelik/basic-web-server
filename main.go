package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/AhmettCelik/web-server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

type interpreter struct {
	Body  string `json:"body"`
	Email string `json:"email"`
}

type user struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Email     string `json:"email"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) requestsCountHandler(w http.ResponseWriter, req *http.Request) {
	res := fmt.Sprintf(`<html>
                            <body>
                                <h1>Welcome, Chirpy Admin</h1>
                                <p>Chirpy has been visited %d times!</p>
                            </body>
                        </html> `,
		cfg.fileserverHits.Load())
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(res))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Swap(0)
	cfg.platform = os.Getenv("PLATFORM")
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Can not use reset method on non-dev environment")
		return
	}

	if err := cfg.db.DeleteUsers(context.Background()); err != nil {
		log.Fatalf("Error deleting users: %v", err)
		respondWithError(w, http.StatusBadRequest, "Error deleting users")
	}
}

func endpointHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "JSON encode error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Write(data)
}

func breakingBadWords(text string) string {
	words := strings.Split(text, " ")
	for i, word := range words {
		switch strings.ToLower(word) {
		case "kerfuffle", "sharbert", "fornax":
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func validatePost(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	post := interpreter{}
	err := decoder.Decode(&post)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(post.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanedBody := breakingBadWords(post.Body)

	response := struct {
		CleanedBody string `json:"cleaned_body"`
	}{
		CleanedBody: cleanedBody,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	inter := interpreter{}
	if err := decoder.Decode(&inter); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
	}

	userDb, err := cfg.db.CreateUser(context.Background(), inter.Email)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid email")
	}

	userJson := user{
		Id:        userDb.ID.String(),
		CreatedAt: userDb.CreatedAt.String(),
		UpdatedAt: userDb.UpdatedAt.String(),
		Email:     userDb.Email,
	}

	respondWithJSON(w, 201, userJson)
}

func main() {
	godotenv.Load()
	var apicfg apiConfig

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error sql.Open: %v", err)
		return
	}

	dbQueries := database.New(db)
	apicfg.db = dbQueries

	serveMuxplier := http.NewServeMux()
	server := http.Server{
		Handler: serveMuxplier,
		Addr:    ":8080",
	}
	serveMuxplier.Handle("/app/", http.StripPrefix("/app", apicfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	serveMuxplier.HandleFunc("GET /api/healthz", endpointHandler)
	serveMuxplier.HandleFunc("GET /admin/metrics", apicfg.requestsCountHandler)
	serveMuxplier.HandleFunc("POST /admin/reset", apicfg.resetHandler)
	serveMuxplier.HandleFunc("POST /api/validate_chirp", validatePost)
	serveMuxplier.HandleFunc("POST /api/users", apicfg.createUserHandler)
	server.ListenAndServe()
}
