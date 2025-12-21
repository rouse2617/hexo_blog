// Package mcp provides MCP (Model Context Protocol) client functionality.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionStatus represents the status of an MCP connection.
type ConnectionStatus string

const (
	StatusOnline  ConnectionStatus = "online"
	StatusOffline ConnectionStatus = "offline"
	StatusError   ConnectionStatus = "error"
)

// Client interface defines MCP client operations.
type Client interface {
	Connect(config ServerConfig) error
	Disconnect() error
	ListTools() ([]Tool, error)
	CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error)
	Status() ConnectionStatus
}

// Manager interface defines MCP connection management operations.
type Manager interface {
	AddServer(config ServerConfig) error
	RemoveServer(serverID string) error
	GetClient(serverID string) (Client, error)
	ListServers() []ServerInfo
	BroadcastStatus() error
}

// ServerConfig represents MCP server configuration.
type ServerConfig struct {
	ID        string
	Name      string
	Transport string // stdio, http
	Command   string
	Args      []string
	Env       map[string]string
	URL       string
	Auth      *AuthConfig
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Type  string // bearer, basic
	Token string
}

// ServerInfo represents MCP server information.
type ServerInfo struct {
	ID     string
	Name   string
	Status ConnectionStatus
	Tools  []Tool
}

// Tool represents an MCP tool.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Readonly    bool
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	Success bool
	Data    interface{}
	Error   string
}

// mcpClient implements the Client interface.
type mcpClient struct {
	config   ServerConfig
	protocol *Protocol
	status   ConnectionStatus
	tools    []Tool
	logger   *zap.Logger

	// For stdio transport
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// For HTTP transport
	httpClient *http.Client

	mu sync.RWMutex
}

// NewClient creates a new MCP client.
func NewClient(logger *zap.Logger) Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &mcpClient{
		protocol:   NewProtocol(),
		status:     StatusOffline,
		logger:     logger,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Connect establishes a connection to the MCP server.
func (c *mcpClient) Connect(config ServerConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = config

	var err error
	switch config.Transport {
	case "stdio":
		err = c.connectStdio()
	case "http":
		err = c.connectHTTP()
	default:
		c.status = StatusError
		return fmt.Errorf("unsupported transport: %s", config.Transport)
	}

	if err != nil {
		c.status = StatusError
		c.logger.Error("Failed to connect to MCP server",
			zap.String("server", config.ID),
			zap.Error(err))
		return err
	}

	c.status = StatusOnline
	c.logger.Info("Connected to MCP server",
		zap.String("server", config.ID),
		zap.String("transport", config.Transport))

	// Fetch tools after connection
	if err := c.fetchTools(); err != nil {
		c.logger.Warn("Failed to fetch tools",
			zap.String("server", config.ID),
			zap.Error(err))
	}

	return nil
}

// connectStdio establishes a stdio connection.
func (c *mcpClient) connectStdio() error {
	cmd := exec.Command(c.config.Command, c.config.Args...)

	// Set environment variables
	if len(c.config.Env) > 0 {
		env := cmd.Environ()
		for k, v := range c.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	var err error
	c.stdin, err = cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stderr, err = cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	c.cmd = cmd

	// Start stderr reader
	go c.readStderr()

	return nil
}

// connectHTTP establishes an HTTP connection.
func (c *mcpClient) connectHTTP() error {
	// For HTTP, we just verify the URL is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	if c.config.Auth != nil && c.config.Auth.Type == "bearer" {
		req.Header.Set("Authorization", "Bearer "+c.config.Auth.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to HTTP server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP server returned error status: %d", resp.StatusCode)
	}

	return nil
}

// Disconnect closes the connection to the MCP server.
func (c *mcpClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusOffline {
		return nil
	}

	var err error
	switch c.config.Transport {
	case "stdio":
		if c.stdin != nil {
			c.stdin.Close()
		}
		if c.stdout != nil {
			c.stdout.Close()
		}
		if c.stderr != nil {
			c.stderr.Close()
		}
		if c.cmd != nil && c.cmd.Process != nil {
			err = c.cmd.Process.Kill()
		}
	case "http":
		// HTTP connections don't need explicit disconnect
	}

	c.status = StatusOffline
	c.logger.Info("Disconnected from MCP server", zap.String("server", c.config.ID))

	return err
}

// ListTools returns the list of available tools.
func (c *mcpClient) ListTools() ([]Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.status != StatusOnline {
		return nil, fmt.Errorf("client is not connected")
	}

	return c.tools, nil
}

// CallTool calls a tool on the MCP server.
func (c *mcpClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error) {
	c.mu.RLock()
	status := c.status
	transport := c.config.Transport
	c.mu.RUnlock()

	if status != StatusOnline {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create request
	req, err := c.protocol.CreateCallToolRequest(toolName, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool call request: %w", err)
	}

	// Serialize request
	reqData, err := c.protocol.SerializeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	// Send request based on transport
	var respData []byte
	switch transport {
	case "stdio":
		respData, err = c.sendStdioRequest(ctx, reqData)
	case "http":
		respData, err = c.sendHTTPRequest(ctx, reqData)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", transport)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Deserialize response
	resp, err := c.protocol.DeserializeResponse(respData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	// Check for error in response
	if resp.Error != nil {
		return &ToolResult{
			Success: false,
			Error:   resp.Error.Message,
		}, nil
	}

	// Parse result
	return &ToolResult{
		Success: true,
		Data:    resp.Result,
	}, nil
}

// Status returns the current connection status.
func (c *mcpClient) Status() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// fetchTools fetches the list of tools from the server.
func (c *mcpClient) fetchTools() error {
	req, err := c.protocol.CreateListToolsRequest()
	if err != nil {
		return err
	}

	reqData, err := c.protocol.SerializeRequest(req)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var respData []byte
	switch c.config.Transport {
	case "stdio":
		respData, err = c.sendStdioRequest(ctx, reqData)
	case "http":
		respData, err = c.sendHTTPRequest(ctx, reqData)
	default:
		return fmt.Errorf("unsupported transport: %s", c.config.Transport)
	}

	if err != nil {
		return err
	}

	resp, err := c.protocol.DeserializeResponse(respData)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("server returned error: %s", resp.Error.Message)
	}

	// Parse tools from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return err
	}

	var listResult ListToolsResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return err
	}

	// Convert to internal Tool type
	c.tools = make([]Tool, len(listResult.Tools))
	for i, mcpTool := range listResult.Tools {
		c.tools[i] = Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			InputSchema: mcpTool.InputSchema,
			Readonly:    false, // Default to false, can be determined from schema
		}
	}

	return nil
}

// sendStdioRequest sends a request via stdio and waits for response.
func (c *mcpClient) sendStdioRequest(ctx context.Context, reqData []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Write request
	if _, err := c.stdin.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}
	if _, err := c.stdin.Write([]byte("\n")); err != nil {
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.stdout)
	respChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			errChan <- err
			return
		}
		respChan <- line
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case resp := <-respChan:
		return resp, nil
	}
}

// sendHTTPRequest sends a request via HTTP and waits for response.
func (c *mcpClient) sendHTTPRequest(ctx context.Context, reqData []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.URL, bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.config.Auth != nil && c.config.Auth.Type == "bearer" {
		req.Header.Set("Authorization", "Bearer "+c.config.Auth.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP server returned error status: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respData, nil
}

// readStderr reads stderr output from the stdio process.
func (c *mcpClient) readStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		c.logger.Debug("MCP server stderr",
			zap.String("server", c.config.ID),
			zap.String("message", scanner.Text()))
	}
}
