
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"monitoring-agent/config"
	pbClient "monitoring-agent/pocketbase"
)

type Agent struct {
	config        *config.Config
	httpClient    *http.Client
	pocketBase    *pbClient.PocketBaseClient
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	
	// Control state
	isMonitoring  bool
	controlMutex  sync.RWMutex
	serverRecord  *pbClient.ServerRecord // Store server record for updates
	currentTicker *time.Ticker           // Current ticker for dynamic interval changes
	tickerMutex   sync.Mutex             // Mutex for ticker operations
}

type SystemMetrics struct {
	AgentID       string    `json:"agent_id"`
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	DiskUsage     float64   `json:"disk_usage"`
	NetworkStats  NetworkStats `json:"network_stats"`
	Uptime        int64     `json:"uptime"`
	GoRoutines    int       `json:"goroutines"`
	Status        string    `json:"status"`
}

type NetworkStats struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	PacketsSent   uint64 `json:"packets_sent"`
	PacketsReceived uint64 `json:"packets_received"`
}

type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	AgentID   string    `json:"agent_id"`
	Version   string    `json:"version"`
}

func New(cfg *config.Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	
	agent := &Agent{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		ctx:          ctx,
		cancel:       cancel,
		isMonitoring: true,
	}

	// Initialize PocketBase client if enabled and configured
	if cfg.PocketBaseEnabled && cfg.PocketBaseURL != "" {
		pbClient, err := pbClient.NewPocketBaseClient(cfg.PocketBaseURL)
		if err != nil {
			log.Printf("Failed to initialize PocketBase client: %v", err)
		} else {
			agent.pocketBase = pbClient
			log.Printf("PocketBase client initialized successfully for %s", cfg.PocketBaseURL)
		}
	} else {
		log.Printf("PocketBase disabled or URL not configured")
	}

	return agent
}

func (a *Agent) Start() error {
	log.Printf("Starting monitoring agent with ID: %s", a.config.AgentID)
	
	// Validate configuration
	if err := a.validateConfiguration(); err != nil {
		log.Printf("Configuration validation failed: %v", err)
		return err
	}
	
	// Initialize or find existing server record
	if err := a.initializeServerRecord(); err != nil {
		log.Printf("Failed to initialize server record: %v", err)
		return err
	}
	
	// Update agent status (optional - don't fail if collection doesn't exist)
	if err := a.updateAgentStatus("running", "Agent started successfully"); err != nil {
		log.Printf("Warning: Failed to update agent status (this is optional): %v", err)
	}
	
	// Start metrics collection
	a.wg.Add(1)
	go a.collectMetrics()
	
	// Start health check server
	a.wg.Add(1)
	go a.startHealthCheckServer()
	
	// Start remote control listener if enabled
	if a.config.RemoteControlEnabled {
		a.wg.Add(1)
		go a.listenForCommands()
	}
	
	// Wait for context cancellation
	<-a.ctx.Done()
	return nil
}

func (a *Agent) validateConfiguration() error {
	// Check basic configuration
	if a.config.AgentID == "" {
		return fmt.Errorf("AGENT_ID is required")
	}
	
	// Check PocketBase configuration if enabled
	if a.config.PocketBaseEnabled {
		if a.config.PocketBaseURL == "" {
			return fmt.Errorf("POCKETBASE_URL is required when POCKETBASE_ENABLED=true")
		}
		if a.config.ServerName == "" {
			return fmt.Errorf("SERVER_NAME is required when POCKETBASE_ENABLED=true")
		}
		if a.config.ServerToken == "" {
			return fmt.Errorf("SERVER_TOKEN is required when POCKETBASE_ENABLED=true")
		}
	}
	
	// Check fallback HTTP configuration if PocketBase is disabled
	if !a.config.PocketBaseEnabled {
		if a.config.ServerURL == "" {
			return fmt.Errorf("SERVER_URL is required when POCKETBASE_ENABLED=false")
		}
		if a.config.APIKey == "" {
			log.Printf("Warning: API_KEY not set for HTTP fallback")
		}
	}
	
	log.Printf("Configuration validation passed")
	return nil
}

func (a *Agent) Stop() {
	log.Println("Stopping monitoring agent...")
	
	// Update agent status
	if err := a.updateAgentStatus("stopped", "Agent stopped by user"); err != nil {
		log.Printf("Failed to update agent status: %v", err)
	}
	
	// Stop current ticker if exists
	a.tickerMutex.Lock()
	if a.currentTicker != nil {
		a.currentTicker.Stop()
	}
	a.tickerMutex.Unlock()
	
	a.cancel()
	a.wg.Wait()
}

func (a *Agent) initializeServerRecord() error {
	if a.pocketBase == nil {
		log.Printf("PocketBase not available, skipping server record initialization")
		return nil
	}

	// Get real hostname and system info
	collector := NewSystemCollector()
	sysInfo := collector.GetSystemInfo()

	// Try to find existing server record by server_id (AgentID)
	existingServer, err := a.pocketBase.GetServerByID(a.config.AgentID)
	if err == nil {
		// Server record exists, use it
		a.serverRecord = existingServer
		log.Printf("Found existing server record for agent %s (ID: %s)", a.config.AgentID, existingServer.ID)
		
		// Check if server is paused initially
		if existingServer.Status == "paused" {
			log.Printf("Server %s is currently paused", a.config.AgentID)
			a.controlMutex.Lock()
			a.isMonitoring = false
			a.controlMutex.Unlock()
		}
		
		return nil
	}

	// Server record doesn't exist, create a new one
	log.Printf("Creating new server record for agent %s", a.config.AgentID)
	
	// Format comprehensive system info
	systemInfoString := fmt.Sprintf("%s %s | %s | Kernel: %s | CPU: %s (%d cores) | RAM: %.1f GB | Go %s | IP: %s", 
		sysInfo.OSName, 
		sysInfo.OSVersion,
		sysInfo.Architecture,
		sysInfo.KernelVersion,
		sysInfo.CPUModel,
		sysInfo.CPUCores,
		float64(sysInfo.TotalRAM)/1024/1024/1024,
		sysInfo.GoVersion,
		sysInfo.IPAddress,
	)
	
	serverRecord := pbClient.ServerRecord{
		ServerID:      a.config.AgentID,
		Name:          a.config.ServerName,
		Hostname:      sysInfo.Hostname,  // Use real hostname
		IPAddress:     sysInfo.IPAddress, // Use real IP address
		OSType:        sysInfo.OSType,    // Use real OS type
		Status:        "up",
		ServerToken:   a.config.ServerToken,
		LastChecked:   pbClient.FlexibleTime{Time: time.Now()},
		Connection:    "connected",
		SystemInfo:    systemInfoString, // Comprehensive system info
		CheckInterval: pbClient.FlexibleInt{Value: int(a.config.CheckInterval.Seconds())}, // Set default check interval
	}

	if err := a.pocketBase.SaveServerMetrics(serverRecord); err != nil {
		return fmt.Errorf("failed to create server record: %v", err)
	}

	// Fetch the created record to get the ID
	createdServer, err := a.pocketBase.GetServerByID(a.config.AgentID)
	if err != nil {
		return fmt.Errorf("failed to fetch created server record: %v", err)
	}

	a.serverRecord = createdServer
	log.Printf("Successfully created server record with ID: %s", a.serverRecord.ID)
	return nil
}

func (a *Agent) checkServerStatus() (bool, time.Duration, error) {
	if a.pocketBase == nil || a.serverRecord == nil {
		return true, a.config.CheckInterval, nil // Default to monitoring if no PocketBase
	}

	// Fetch current server record to check status and interval
	currentServer, err := a.pocketBase.GetServerByID(a.config.AgentID)
	if err != nil {
		log.Printf("Failed to fetch server status: %v", err)
		return true, a.config.CheckInterval, nil // Continue monitoring on error
	}

	// Update our local copy
	a.serverRecord = currentServer
	
	// Get check interval from server record, fallback to config default
	checkInterval := a.config.CheckInterval
	if currentServer.CheckInterval.Value > 0 {
		checkInterval = time.Duration(currentServer.CheckInterval.Value) * time.Second
		log.Printf("Using check interval from server record: %v", checkInterval)
	}
	
	// Check if server is paused
	isPaused := currentServer.Status == "paused"
	if isPaused {
		log.Printf("Server %s is paused, skipping monitoring", a.config.AgentID)
		a.controlMutex.Lock()
		a.isMonitoring = false
		a.controlMutex.Unlock()
	} else {
		a.controlMutex.Lock()
		wasMonitoring := a.isMonitoring
		a.isMonitoring = true
		a.controlMutex.Unlock()
		
		if !wasMonitoring {
			log.Printf("Server %s monitoring resumed", a.config.AgentID)
		}
	}
	
	return !isPaused, checkInterval, nil
}

func (a *Agent) collectMetrics() {
	defer a.wg.Done()
	
	// Start with default interval
	currentInterval := a.config.CheckInterval
	a.tickerMutex.Lock()
	a.currentTicker = time.NewTicker(currentInterval)
	ticker := a.currentTicker
	a.tickerMutex.Unlock()
	
	defer ticker.Stop()
	
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			// Check server status and get current interval
			shouldMonitor, newInterval, err := a.checkServerStatus()
			if err != nil {
				log.Printf("Error checking server status: %v", err)
			}
			
			// Update ticker if interval changed
			if newInterval != currentInterval {
				log.Printf("Check interval changed from %v to %v", currentInterval, newInterval)
				currentInterval = newInterval
				
				a.tickerMutex.Lock()
				ticker.Stop()
				a.currentTicker = time.NewTicker(currentInterval)
				ticker = a.currentTicker
				a.tickerMutex.Unlock()
			}
			
			if !shouldMonitor {
				continue // Skip this cycle if server is paused
			}
			
			a.controlMutex.RLock()
			if !a.isMonitoring {
				a.controlMutex.RUnlock()
				continue
			}
			a.controlMutex.RUnlock()
			
			// Collect server metrics for the servers collection
			serverMetrics := a.gatherServerMetrics()
			
			// Collect detailed server metrics for the server_metrics collection
			detailedMetrics := a.gatherDetailedServerMetrics()
			
			// Update server record instead of creating new one
			if err := a.updateServerRecord(serverMetrics); err != nil {
				log.Printf("Failed to update server record: %v", err)
			} else {
				log.Printf("Successfully updated server record at %s", time.Now().Format(time.RFC3339))
			}
			
			// Send detailed metrics to the server_metrics collection
			if err := a.sendDetailedServerMetrics(detailedMetrics); err != nil {
				log.Printf("Failed to send detailed server metrics: %v", err)
			} else {
				log.Printf("Successfully sent detailed server metrics at %s", time.Now().Format(time.RFC3339))
			}
			
			// Handle Docker monitoring if enabled
			if serverMetrics.Docker.Value {
				log.Printf("Docker is available, collecting Docker metrics...")
				
				// Collect Docker container records
				dockerRecords := a.gatherDockerContainers()
				if err := a.sendDockerRecords(dockerRecords); err != nil {
					log.Printf("Failed to send Docker records: %v", err)
				} else if len(dockerRecords) > 0 {
					log.Printf("Successfully sent %d Docker records at %s", len(dockerRecords), time.Now().Format(time.RFC3339))
				}
				
				// Collect Docker metrics
				dockerMetrics := a.gatherDockerMetrics()
				if err := a.sendDockerMetrics(dockerMetrics); err != nil {
					log.Printf("Failed to send Docker metrics: %v", err)
				} else if len(dockerMetrics) > 0 {
					log.Printf("Successfully sent %d Docker metrics at %s", len(dockerMetrics), time.Now().Format(time.RFC3339))
				}
			} else {
				log.Printf("Docker is not available on this server, skipping Docker monitoring")
			}
		}
	}
}

func (a *Agent) updateServerRecord(serverMetrics pbClient.ServerRecord) error {
	if a.pocketBase == nil || a.serverRecord == nil {
		return fmt.Errorf("no PocketBase client or server record available")
	}

	// Update the existing server record
	return a.pocketBase.UpdateServerStatus(a.serverRecord.ID, serverMetrics)
}

func (a *Agent) listenForCommands() {
	defer a.wg.Done()
	
	ticker := time.NewTicker(a.config.CommandCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.checkForCommands(); err != nil {
				log.Printf("Warning: Failed to check for commands (this is optional): %v", err)
			}
		}
	}
}

func (a *Agent) checkForCommands() error {
	// Check PocketBase for commands
	if a.pocketBase != nil {
		commands, err := a.pocketBase.GetPendingCommands(a.config.AgentID)
		if err != nil {
			return err
		}
		
		for _, cmd := range commands {
			parameters := make(map[string]string)
			// Fix: cmd.Parameters is already a string from PocketBase
			if cmd.Parameters != "" {
				if err := json.Unmarshal([]byte(cmd.Parameters), &parameters); err != nil {
					log.Printf("Failed to parse command parameters: %v", err)
					continue
				}
			}
			
			if err := a.executeCommand(cmd.Command, parameters); err != nil {
				log.Printf("Failed to execute command %s: %v", cmd.Command, err)
				continue
			}
			
			// Fix: Use cmd.ID which now exists in the CommandRecord
			if err := a.pocketBase.MarkCommandExecuted(cmd.ID); err != nil {
				log.Printf("Failed to mark command as executed: %v", err)
			}
		}
	}
	
	return nil
}

func (a *Agent) executeCommand(command string, parameters map[string]string) error {
	log.Printf("Executing command: %s with parameters: %v", command, parameters)
	
	switch command {
	case "start":
		return a.startMonitoring()
	case "stop":
		return a.stopMonitoring()
	case "restart":
		if err := a.stopMonitoring(); err != nil {
			return err
		}
		return a.startMonitoring()
	case "config_update":
		return a.updateConfiguration(parameters)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func (a *Agent) startMonitoring() error {
	a.controlMutex.Lock()
	defer a.controlMutex.Unlock()
	
	a.isMonitoring = true
	log.Println("Monitoring started via remote command")
	return a.updateAgentStatus("running", "Monitoring started via remote command")
}

func (a *Agent) stopMonitoring() error {
	a.controlMutex.Lock()
	defer a.controlMutex.Unlock()
	
	a.isMonitoring = false
	log.Println("Monitoring stopped via remote command")
	return a.updateAgentStatus("paused", "Monitoring stopped via remote command")
}

func (a *Agent) updateConfiguration(parameters map[string]string) error {
	// Update configuration based on parameters
	// This is a simplified implementation
	log.Printf("Configuration update requested with parameters: %v", parameters)
	return a.updateAgentStatus("running", "Configuration updated via remote command")
}

func (a *Agent) updateAgentStatus(status, message string) error {
	// Update via PocketBase
	if a.pocketBase != nil {
		statusRecord := pbClient.AgentStatusRecord{
			AgentID:  a.config.AgentID,
			Status:   status,
			LastSeen: time.Now(),
			Version:  "1.0.0",
			Message:  message,
		}
		
		if err := a.pocketBase.UpdateAgentStatus(statusRecord); err != nil {
			// Don't treat this as a fatal error, just log it
			log.Printf("Warning: Failed to update status via PocketBase: %v", err)
			return err
		}
	}
	
	return nil
}

func (a *Agent) gatherSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	collector := NewSystemCollector()
	uptimeSeconds := collector.GetSystemUptime()
	
	return SystemMetrics{
		AgentID:     a.config.AgentID,
		Timestamp:   time.Now(),
		CPUUsage:    collector.GetCPUUsage(),
		MemoryUsage: float64(m.Alloc) / 1024 / 1024, // MB
		DiskUsage:   a.getDiskUsage(),
		NetworkStats: NetworkStats{
			BytesSent:       0, // Would implement actual network stats
			BytesReceived:   0,
			PacketsSent:     0,
			PacketsReceived: 0,
		},
		Uptime:     uptimeSeconds,
		GoRoutines: runtime.NumGoroutine(),
		Status:     "healthy",
	}
}

func (a *Agent) getCPUUsage() float64 {
	collector := NewSystemCollector()
	return collector.GetCPUUsage()
}

func (a *Agent) getDiskUsage() float64 {
	collector := NewSystemCollector()
	_, _, percentage := collector.GetDiskUsage()
	return percentage
}

func (a *Agent) sendMetricsHTTP(metrics SystemMetrics) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}
	
	url := fmt.Sprintf("%s/api/metrics", a.config.ServerURL)
	req, err := http.NewRequestWithContext(a.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)
	req.Header.Set("X-Agent-ID", a.config.AgentID)
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	log.Printf("Successfully sent metrics via HTTP at %s", metrics.Timestamp.Format(time.RFC3339))
	return nil
}

func (a *Agent) startHealthCheckServer() {
	defer a.wg.Done()
	
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.healthHandler)
	mux.HandleFunc("/status", a.statusHandler)
	mux.HandleFunc("/control/start", a.controlStartHandler)
	mux.HandleFunc("/control/stop", a.controlStopHandler)
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.HealthCheckPort),
		Handler: mux,
	}
	
	go func() {
		log.Printf("Health check server starting on port %d", a.config.HealthCheckPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health check server error: %v", err)
		}
	}()
	
	<-a.ctx.Done()
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Health check server shutdown error: %v", err)
	}
}

func (a *Agent) healthHandler(w http.ResponseWriter, r *http.Request) {
	health := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		AgentID:   a.config.AgentID,
		Version:   "1.0.0",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (a *Agent) statusHandler(w http.ResponseWriter, r *http.Request) {
	metrics := a.gatherSystemMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (a *Agent) controlStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if err := a.startMonitoring(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (a *Agent) controlStopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if err := a.stopMonitoring(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}