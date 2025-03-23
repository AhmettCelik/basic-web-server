package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type posts struct {
	Body string `json:"body"`
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
	post := posts{}
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

func main() {
	var apicfg apiConfig
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
	server.ListenAndServe()
}
