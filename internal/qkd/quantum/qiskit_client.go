package quantum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// QiskitConfig holds IBM Qiskit Runtime API configuration
type QiskitConfig struct {
	// IBM Cloud API Key
	APIKey string

	// IBM Cloud CRN (Cloud Resource Name)
	CRN string

	// Base URL for IBM Quantum API
	BaseURL string

	// Backend name (e.g., "ibmq_qasm_simulator", "ibm_kyoto")
	BackendName string

	// HTTP client with timeout
	HTTPClient *http.Client
}

// QiskitClient handles IBM Qiskit Runtime API interactions
type QiskitClient struct {
	config      *QiskitConfig
	accessToken string
	tokenExpiry time.Time
}

// QiskitJob represents a quantum job
type QiskitJob struct {
	ID        string    `json:"id"`
	Backend   string    `json:"backend"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created"`
	Results   *QiskitResult
}

// QiskitResult represents job execution results
type QiskitResult struct {
	Counts      map[string]int `json:"counts"`
	Success     bool           `json:"success"`
	StatusMsg   string         `json:"status"`
	JobID       string         `json:"job_id"`
	ExecutionTime float64      `json:"execution_time"`
}

// QiskitCircuit represents an OpenQASM circuit
type QiskitCircuit struct {
	QASM    string `json:"qasm"`
	Shots   int    `json:"shots"`
	Backend string `json:"backend"`
}

// IBM Quantum API endpoints
const (
	DefaultQiskitURL = "https://api.quantum-computing.ibm.com"
	TokenEndpoint    = "/api/auth/login"
	JobsEndpoint     = "/api/Network/ibm-q/Groups/open/Projects/main/Jobs"
	BackendsEndpoint = "/api/Network/ibm-q/Groups/open/Projects/main/devices"
)

// Job status constants
const (
	JobStatusQueued    = "QUEUED"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
	JobStatusCancelled = "CANCELLED"
)

// NewQiskitClient creates a new Qiskit API client
func NewQiskitClient(config *QiskitConfig) (*QiskitClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("IBM Cloud API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = DefaultQiskitURL
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	client := &QiskitClient{
		config: config,
	}

	// Authenticate immediately
	if err := client.authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return client, nil
}

// authenticate obtains an access token from IBM Cloud
func (c *QiskitClient) authenticate() error {
	url := c.config.BaseURL + TokenEndpoint

	payload := map[string]string{
		"apiToken": c.config.APIKey,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var result struct {
		ID          string    `json:"id"`
		TTL         int       `json:"ttl"`
		Created     time.Time `json:"created"`
		AccessToken string    `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.TTL) * time.Second)

	return nil
}

// ensureAuthenticated checks if token is valid and refreshes if needed
func (c *QiskitClient) ensureAuthenticated() error {
	if time.Now().After(c.tokenExpiry.Add(-5 * time.Minute)) {
		return c.authenticate()
	}
	return nil
}

// SubmitJob submits a quantum circuit for execution
func (c *QiskitClient) SubmitJob(circuit *QiskitCircuit) (*QiskitJob, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, err
	}

	url := c.config.BaseURL + JobsEndpoint

	payload := map[string]interface{}{
		"qasm":    circuit.QASM,
		"shots":   circuit.Shots,
		"backend": circuit.Backend,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("job submission failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var job QiskitJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// GetJobStatus retrieves the status of a quantum job
func (c *QiskitClient) GetJobStatus(jobID string) (*QiskitJob, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s/%s", c.config.BaseURL, JobsEndpoint, jobID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get job status failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var job QiskitJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// WaitForJob waits for a job to complete with polling
func (c *QiskitClient) WaitForJob(jobID string, maxWaitTime time.Duration) (*QiskitJob, error) {
	pollInterval := 2 * time.Second
	timeout := time.After(maxWaitTime)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("job %s timed out after %v", jobID, maxWaitTime)

		case <-ticker.C:
			job, err := c.GetJobStatus(jobID)
			if err != nil {
				return nil, err
			}

			switch job.Status {
			case JobStatusCompleted:
				return job, nil
			case JobStatusFailed:
				return job, fmt.Errorf("job %s failed", jobID)
			case JobStatusCancelled:
				return job, fmt.Errorf("job %s was cancelled", jobID)
			// Continue polling for QUEUED and RUNNING
			}
		}
	}
}

// GetJobResult retrieves the results of a completed job
func (c *QiskitClient) GetJobResult(jobID string) (*QiskitResult, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s/%s/results", c.config.BaseURL, JobsEndpoint, jobID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get job result failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var result QiskitResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CancelJob cancels a running or queued job
func (c *QiskitClient) CancelJob(jobID string) error {
	if err := c.ensureAuthenticated(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s/%s/cancel", c.config.BaseURL, JobsEndpoint, jobID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel job failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// ListBackends retrieves available quantum backends
func (c *QiskitClient) ListBackends() ([]map[string]interface{}, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, err
	}

	url := c.config.BaseURL + BackendsEndpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list backends failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var backends []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&backends); err != nil {
		return nil, err
	}

	return backends, nil
}

// ExecuteCircuitSync executes a circuit synchronously and returns results
func (c *QiskitClient) ExecuteCircuitSync(circuit *QiskitCircuit, maxWaitTime time.Duration) (*QiskitResult, error) {
	// Submit job
	job, err := c.SubmitJob(circuit)
	if err != nil {
		return nil, fmt.Errorf("job submission failed: %w", err)
	}

	// Wait for completion
	completedJob, err := c.WaitForJob(job.ID, maxWaitTime)
	if err != nil {
		return nil, fmt.Errorf("job execution failed: %w", err)
	}

	// Get results
	result, err := c.GetJobResult(completedJob.ID)
	if err != nil {
		return nil, fmt.Errorf("result retrieval failed: %w", err)
	}

	return result, nil
}
