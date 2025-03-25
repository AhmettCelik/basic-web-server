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

	"github.com/AhmettCelik/web-server/internal/auth"
	"github.com/AhmettCelik/web-server/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

type interpreter struct {
	Body     string `json:"body"`
	Email    string `json:"email"`
	UserId   string `json:"user_id"`
	Password string `json:"password"`
}

type user struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Email     string `json:"email"`
	password  string
}

type chirp struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	UserId    string `json:"user_id"`
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

func (cfg *apiConfig) validatePost(w http.ResponseWriter, req *http.Request) {
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
	userId, err := uuid.Parse(post.UserId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "User id can not parsed to UUID")
		return
	}

	params := database.CreateChirpParams{
		UserID: userId,
		Body:   cleanedBody,
	}

	chirpDb, err := cfg.db.CreateChirp(context.Background(), params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong see the log")
		log.Fatalf("Error creating chirp: %v", err)
		return
	}

	chirpJson := chirp{
		Id:        chirpDb.ID.String(),
		CreatedAt: chirpDb.CreatedAt.String(),
		UpdatedAt: chirpDb.UpdatedAt.String(),
		Body:      cleanedBody,
		UserId:    userId.String(),
	}

	respondWithJSON(w, 201, chirpJson)
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	inter := interpreter{}
	if err := decoder.Decode(&inter); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
	}

	password := inter.Password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("Error hashing password: %v", err)
		respondWithError(w, http.StatusBadRequest, "Password could not be hashed")
		return
	}

	params := database.CreateUserParams{
		Email:          inter.Email,
		HashedPassword: hashedPassword,
	}

	userDb, err := cfg.db.CreateUser(context.Background(), params)
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

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	chirpsDb, err := cfg.db.GetChirps(context.Background())
	if err != nil {
		log.Fatalf("Error getting chirps: %v", err)
		respondWithError(w, http.StatusBadRequest, "Error getting chirps")
		return
	}

	var chirpsJson []chirp
	for _, chirpDb := range chirpsDb {
		chirpJson := chirp{
			Id:        chirpDb.ID.String(),
			CreatedAt: chirpDb.CreatedAt.String(),
			UpdatedAt: chirpDb.UpdatedAt.String(),
			Body:      chirpDb.Body,
			UserId:    chirpDb.UserID.String(),
		}
		chirpsJson = append(chirpsJson, chirpJson)
	}

	respondWithJSON(w, http.StatusOK, chirpsJson)
}

func (cfg *apiConfig) getChirpById(w http.ResponseWriter, req *http.Request) {
	chirpId, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		log.Fatalf("Error parsing uuid string: %v", err)
		respondWithError(w, http.StatusBadRequest, "Cant parse uuid string")
		return
	}

	chirpDb, err := cfg.db.GetChirpById(context.Background(), chirpId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	chirpJson := chirp{
		Id:        chirpDb.ID.String(),
		CreatedAt: chirpDb.CreatedAt.String(),
		UpdatedAt: chirpDb.UpdatedAt.String(),
		Body:      chirpDb.Body,
		UserId:    chirpDb.UserID.String(),
	}

	respondWithJSON(w, 200, chirpJson)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	inter := interpreter{}
	if err := decoder.Decode(&inter); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
	}
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
	serveMuxplier.HandleFunc("POST /api/users", apicfg.createUserHandler)
	serveMuxplier.HandleFunc("POST /api/chirps", apicfg.validatePost)
	serveMuxplier.HandleFunc("GET /api/chirps", apicfg.getChirps)
	serveMuxplier.HandleFunc("GET /api/chirps/{chirpID}", apicfg.getChirpById)
	serveMuxplier.HandleFunc("POST /api/login", apicfg.loginHandler)
	server.ListenAndServe()
}
