package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	log.Printf("got / request\n")
	io.WriteString(w, "Hello!\n")
}
func getHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /health_check request\n")
	io.WriteString(w, "OK\n")
}

func setupRoutes() {
	http.HandleFunc("/", getRoot)
	http.HandleFunc("/health", getHealth)
}

func startServer(listenPort string) {
	setupRoutes()

	err := http.ListenAndServe(fmt.Sprintf(":%v", listenPort), nil)
	if err != nil {
		log.Fatal(err)
	}
}
