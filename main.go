package main

import (
	"log"
	"os"

	"github.com/valyala/fasthttp"

	"pension-engine/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Pension engine starting on port %s", port)
	if err := fasthttp.ListenAndServe(":"+port, handler.HandleCalculation); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
