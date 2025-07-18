
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"monitoring-agent/agent"
	"monitoring-agent/config"
)

func main() {
	// Set up logging to both stdout and file
	logFile, err := os.OpenFile("/var/log/monitoring-agent/monitoring-agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If we can't write to the log file, just use stdout
		log.Printf("Warning: Could not open log file: %v, using stdout only", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Println("=== Starting monitoring agent ===")
	log.Printf("PID: %d", os.Getpid())
	log.Printf("Working directory: %s", os.Getenv("PWD"))
	log.Printf("User: %s", os.Getenv("USER"))

	// Load configuration with detailed error logging
	log.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Printf("FATAL: Failed to load configuration: %v", err)
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Configuration loaded successfully:")
	log.Printf("  - Agent ID: %s", cfg.AgentID)
	log.Printf("  - PocketBase Enabled: %t", cfg.PocketBaseEnabled)
	log.Printf("  - PocketBase URL: %s", cfg.PocketBaseURL)
	log.Printf("  - Server Name: %s", cfg.ServerName)
	log.Printf("  - Check Interval: %v", cfg.CheckInterval)
	log.Printf("  - Health Check Port: %d", cfg.HealthCheckPort)

	// Create and start the monitoring agent
	log.Println("Creating monitoring agent...")
	monitoringAgent := agent.New(cfg)
	
	// Start monitoring in a goroutine
	go func() {
		log.Println("Starting monitoring agent...")
		if err := monitoringAgent.Start(); err != nil {
			log.Printf("FATAL: Failed to start monitoring agent: %v", err)
			fmt.Fprintf(os.Stderr, "Agent start error: %v\n", err)
			os.Exit(1)
		}
	}()

	log.Println("Monitoring agent started successfully")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down monitoring agent...")
	monitoringAgent.Stop()
	log.Println("Monitoring agent stopped")
}