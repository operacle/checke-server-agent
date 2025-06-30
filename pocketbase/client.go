package pocketbase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PocketBaseClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPocketBaseClient(baseURL string) (*PocketBaseClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("PocketBase URL cannot be empty")
	}

	return &PocketBaseClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *PocketBaseClient) GetBaseURL() string {
	return c.baseURL
}

func (c *PocketBaseClient) TestConnection() error {
	resp, err := c.httpClient.Get(c.baseURL + "/api/health")
	if err != nil {
		return fmt.Errorf("connection test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *PocketBaseClient) GetServerByID(serverID string) (*ServerRecord, error) {
	url := fmt.Sprintf("%s/api/collections/servers/records?filter=server_id='%s'", c.baseURL, serverID)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server not found, status: %d", resp.StatusCode)
	}

	var response struct {
		Items []ServerRecord `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("server record not found")
	}

	server := &response.Items[0]
	return server, nil
}

func (c *PocketBaseClient) SaveServerMetrics(server ServerRecord) error {
	jsonData, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server record: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/servers/records", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to save server metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create record, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *PocketBaseClient) UpdateServerStatus(recordID string, server ServerRecord) error {
	jsonData, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server record: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/servers/records/%s", c.baseURL, recordID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update server status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update record, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *PocketBaseClient) SaveServerMetricsRecord(metrics ServerMetricsRecord) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal server metrics: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/server_metrics/records", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to save server metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save server metrics, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateAgentStatus now updates the agent_status field in the servers collection
func (c *PocketBaseClient) UpdateAgentStatus(status AgentStatusRecord) error {
	// Find the server record by agent_id (server_id)
	server, err := c.GetServerByID(status.AgentID)
	if err != nil {
		return fmt.Errorf("failed to find server record: %v", err)
	}

	// Update only the agent_status field in the server record
	updateData := map[string]interface{}{
		"agent_status": status.Status,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal agent status update: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/servers/records/%s", c.baseURL, server.ID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create update request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update agent status, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *PocketBaseClient) GetPendingCommands(agentID string) ([]CommandRecord, error) {
	url := fmt.Sprintf("%s/api/collections/commands/records?filter=agent_id='%s'&&executed=false", c.baseURL, agentID)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []CommandRecord{}, nil // Return empty slice if collection doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get commands, status: %d", resp.StatusCode)
	}

	var response struct {
		Items []CommandRecord `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return response.Items, nil
}

func (c *PocketBaseClient) MarkCommandExecuted(commandID string) error {
	data := map[string]bool{"executed": true}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal command update: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/commands/records/%s", c.baseURL, commandID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to mark command executed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mark command executed, status: %d", resp.StatusCode)
	}

	return nil
}

// SaveDockerRecord saves a Docker container record
func (c *PocketBaseClient) SaveDockerRecord(docker DockerRecord) error {
	jsonData, err := json.Marshal(docker)
	if err != nil {
		return fmt.Errorf("failed to marshal docker record: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/dockers/records", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to save docker record: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create docker record, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SaveDockerMetricsRecord saves Docker container metrics
func (c *PocketBaseClient) SaveDockerMetricsRecord(metrics DockerMetricsRecord) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal docker metrics: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/docker_metrics/records", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to save docker metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save docker metrics, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetDockerByID gets a Docker container record by docker_id
func (c *PocketBaseClient) GetDockerByID(dockerID string) (*DockerRecord, error) {
	url := fmt.Sprintf("%s/api/collections/dockers/records?filter=docker_id='%s'", c.baseURL, dockerID)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker record: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker record not found, status: %d", resp.StatusCode)
	}

	var response struct {
		Items []DockerRecord `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("docker record not found")
	}

	docker := &response.Items[0]
	return docker, nil
}

// UpdateDockerRecord updates an existing Docker record
func (c *PocketBaseClient) UpdateDockerRecord(recordID string, docker DockerRecord) error {
	jsonData, err := json.Marshal(docker)
	if err != nil {
		return fmt.Errorf("failed to marshal docker record: %v", err)
	}

	url := fmt.Sprintf("%s/api/collections/dockers/records/%s", c.baseURL, recordID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update docker record: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update docker record, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}