// Package main is the entry point for the container registry server.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"cyp-registry/internal/common"
	"cyp-registry/internal/dao"
	"cyp-registry/internal/gateway"
	"cyp-registry/internal/version"

	"go.uber.org/zap"
)

const (
	author  = "CYP"
	email   = "nasDSSCYP@outlook.com"
	appName = "CYP-Registry"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	dataPath := flag.String("data", "./data", "Path to data directory")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version and exit if requested
	if *showVersion {
		fmt.Printf("%s %s\n", appName, version.GetFullVersion())
		os.Exit(0)
	}

	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Print copyright information
	printCopyright(logger)

	// Ensure data directory exists
	if err := os.MkdirAll(*dataPath, 0755); err != nil {
		logger.Fatal("Failed to create data directory", zap.Error(err))
	}

	// Initialize database
	dbPath := filepath.Join(*dataPath, "registry.db")
	if err := dao.InitDB(dbPath, logger); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer dao.CloseDB()

	logger.Info("Database initialized", zap.String("path", dbPath))

	// Load configuration
	config, err := common.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize gateway logger
	gateway.InitLogger(logger)

	// Create and start router
	router := gateway.NewRouter(config)

	// Start server
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	logger.Info("Starting server",
		zap.String("address", addr),
		zap.String("version", version.GetVersion()),
	)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := router.Engine().Run(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-quit
	logger.Info("Shutting down server...")
}

// initLogger initializes the zap logger.
func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	return config.Build()
}

// printCopyright prints copyright information at startup.
func printCopyright(logger *zap.Logger) {
	fmt.Println("========================================")
	fmt.Printf("  %s v%s\n", appName, version.GetVersion())
	fmt.Println("========================================")
	fmt.Printf("  Copyright © 2024 %s. All rights reserved.\n", author)
	fmt.Printf("  Author: %s\n", author)
	fmt.Printf("  Contact: %s\n", email)
	fmt.Println("========================================")
	fmt.Println()

	logger.Info("Application started",
		zap.String("app", appName),
		zap.String("version", version.GetVersion()),
		zap.String("author", author),
	)
}
