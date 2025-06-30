
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"monitoring-agent/agent"
	"monitoring-agent/config"
)

func main() {
	log.Println("Starting monitoring agent...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start the monitoring agent
	monitoringAgent := agent.New(cfg)
	
	// Start monitoring in a goroutine
	go func() {
		if err := monitoringAgent.Start(); err != nil {
			log.Fatalf("Failed to start monitoring agent: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down monitoring agent...")
	monitoringAgent.Stop()
	log.Println("Monitoring agent stopped")
}
