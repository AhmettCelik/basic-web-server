package main

import (
	"net/http"
)

func endpointHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	serveMuxplier := http.NewServeMux()
	server := http.Server{
		Handler: serveMuxplier,
		Addr:    ":8080",
	}
	serveMuxplier.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	serveMuxplier.HandleFunc("/healthz", endpointHandler)
	server.ListenAndServe()
}
