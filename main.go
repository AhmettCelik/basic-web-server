package main

import "net/http"

func main() {
	serveMuxplier := http.NewServeMux()
	server := http.Server{
		Handler: serveMuxplier,
		Addr:    ":8080",
	}
	serveMuxplier.Handle("/", http.FileServer(http.Dir(".")))
	server.ListenAndServe()
}
