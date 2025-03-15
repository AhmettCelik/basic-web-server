package main

import "net/http"

func main() {
	serveMuxplier := http.ServeMux{}
	server := http.Server{
		Handler: &serveMuxplier,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
