package main

import (
	"log"
	"os"
	"runtime/debug"

	"github.com/valyala/fasthttp"

	"pension-engine/internal/handler"
)

func main() {
	// Reduce GC frequency: less CPU spent on GC = more throughput under load.
	// Default GOGC=100 (GC when heap doubles). Setting high means GC runs rarely.
	// GOMEMLIMIT provides a safety net so memory doesn't grow unbounded.
	debug.SetGCPercent(2000)
	debug.SetMemoryLimit(512 * 1024 * 1024) // 512 MiB soft limit

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &fasthttp.Server{
		Handler:          handler.HandleCalculation,
		DisableKeepalive: false,
		ReadBufferSize:   8192,
		WriteBufferSize:  8192,
	}

	log.Printf("Pension engine starting on port %s", port)
	if err := server.ListenAndServe(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
