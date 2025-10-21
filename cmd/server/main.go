package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"jsondrop/internal/api"
	"jsondrop/internal/config"
	"jsondrop/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting JSONDrop server...")
	log.Printf("Port: %s", cfg.Port)
	log.Printf("DB Base Directory: %s", cfg.DBBaseDir)
	log.Printf("Catalog DB Path: %s", cfg.CatalogDBPath)
	log.Printf("CORS Origins: %v", cfg.CORSOrigins)
	log.Printf("Default Quota: %d MB", cfg.DefaultQuotaMB)
	log.Printf("Expiry Days: %d", cfg.ExpiryDays)
	log.Printf("Expiry Check Interval: %v", cfg.ExpiryCheckInterval)

	// Initialize catalog database
	catalog, err := database.NewCatalogDB(cfg.CatalogDBPath, cfg.DBBaseDir, cfg.DefaultQuotaMB)
	if err != nil {
		log.Fatalf("Failed to initialize catalog database: %v", err)
	}
	defer catalog.Close()

	log.Println("Catalog database initialized successfully")

	// Create API handler
	handler := api.NewHandler(catalog)

	// Create router
	router := api.NewRouter(handler, catalog, cfg.CORSOrigins)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		if err := server.Close(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	log.Println("Server stopped")
}
