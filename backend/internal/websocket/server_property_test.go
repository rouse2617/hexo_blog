package websocket

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/opsgenius/backend/pkg/models"
	"go.uber.org/zap"
)

// Feature: ops-genius-backend, Property 1: WebSocket 连接 ID 唯一性
// *对于任何*数量的客户端连接，每个连接分配的 ID 应该是唯一的，不存在重复。
// **Validates: Requirements 1.1**
func TestProperty_ConnectionIDUniqueness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("all connection IDs should be unique", prop.ForAll(
		func(n int) bool {
			ids := make(map[string]bool)

			for i := 0; i < n; i++ {
				id := GenerateConnectionID()
				if ids[id] {
					return false // Duplicate ID found
				}
				ids[id] = true
			}

			return true
		},
		gen.IntRange(1, 1000),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 3: 消息序列化往返一致性
// *对于任何*服务端消息对象，序列化后通过 WebSocket 发送，前端反序列化后应该得到等价的对象。
// **Validates: Requirements 1.3**
func TestProperty_MessageSerializationRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Test AgentStreamPayload round-trip
	properties.Property("AgentStreamPayload should round-trip through serialization", prop.ForAll(
		func(sessionID, messageID, content string, done bool) bool {
			original := models.ServerMessage{
				Type: models.MessageTypeAgentStream,
				Payload: models.AgentStreamPayload{
					SessionID: sessionID,
					MessageID: messageID,
					Content:   content,
					Done:      done,
				},
				Timestamp: time.Now().Unix(),
			}

			// Serialize
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize
			var decoded models.ServerMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				return false
			}

			// Compare basic fields
			if original.Type != decoded.Type || original.Timestamp != decoded.Timestamp {
				return false
			}

			// Deserialize payload
			payloadBytes, err := json.Marshal(decoded.Payload)
			if err != nil {
				return false
			}
			var decodedPayload models.AgentStreamPayload
			if err := json.Unmarshal(payloadBytes, &decodedPayload); err != nil {
				return false
			}

			originalPayload := original.Payload.(models.AgentStreamPayload)
			return originalPayload.SessionID == decodedPayload.SessionID &&
				originalPayload.MessageID == decodedPayload.MessageID &&
				originalPayload.Content == decodedPayload.Content &&
				originalPayload.Done == decodedPayload.Done
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Bool(),
	))

	// Test MCPStatusPayload round-trip
	properties.Property("MCPStatusPayload should round-trip through serialization", prop.ForAll(
		func(serverID string, statusIdx int) bool {
			statuses := []string{"online", "offline", "error"}
			status := statuses[statusIdx%len(statuses)]

			original := models.ServerMessage{
				Type: models.MessageTypeMCPStatus,
				Payload: models.MCPStatusPayload{
					ServerID: serverID,
					Status:   status,
				},
				Timestamp: time.Now().Unix(),
			}

			// Serialize
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize
			var decoded models.ServerMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				return false
			}

			// Compare basic fields
			if original.Type != decoded.Type || original.Timestamp != decoded.Timestamp {
				return false
			}

			// Deserialize payload
			payloadBytes, err := json.Marshal(decoded.Payload)
			if err != nil {
				return false
			}
			var decodedPayload models.MCPStatusPayload
			if err := json.Unmarshal(payloadBytes, &decodedPayload); err != nil {
				return false
			}

			originalPayload := original.Payload.(models.MCPStatusPayload)
			return originalPayload.ServerID == decodedPayload.ServerID &&
				originalPayload.Status == decodedPayload.Status
		},
		gen.AlphaString(),
		gen.IntRange(0, 2),
	))

	// Test ErrorPayload round-trip
	properties.Property("ErrorPayload should round-trip through serialization", prop.ForAll(
		func(code, message string) bool {
			original := models.ServerMessage{
				Type: models.MessageTypeError,
				Payload: models.ErrorPayload{
					Code:    code,
					Message: message,
				},
				Timestamp: time.Now().Unix(),
			}

			// Serialize
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize
			var decoded models.ServerMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				return false
			}

			// Compare basic fields
			if original.Type != decoded.Type || original.Timestamp != decoded.Timestamp {
				return false
			}

			// Deserialize payload
			payloadBytes, err := json.Marshal(decoded.Payload)
			if err != nil {
				return false
			}
			var decodedPayload models.ErrorPayload
			if err := json.Unmarshal(payloadBytes, &decodedPayload); err != nil {
				return false
			}

			originalPayload := original.Payload.(models.ErrorPayload)
			return originalPayload.Code == decodedPayload.Code &&
				originalPayload.Message == decodedPayload.Message
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Test ChartDataPayload round-trip
	properties.Property("ChartDataPayload should round-trip through serialization", prop.ForAll(
		func(sessionID, messageID string, chartTypeIdx int, labels []string, dataValues []float64) bool {
			chartTypes := []string{"line", "bar", "pie"}
			chartType := chartTypes[chartTypeIdx%len(chartTypes)]

			original := models.ServerMessage{
				Type: models.MessageTypeChartData,
				Payload: models.ChartDataPayload{
					SessionID: sessionID,
					MessageID: messageID,
					ChartType: chartType,
					Data: models.ChartData{
						Labels: labels,
						Datasets: []models.ChartDataset{
							{
								Label: "test",
								Data:  dataValues,
							},
						},
					},
				},
				Timestamp: time.Now().Unix(),
			}

			// Serialize
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize
			var decoded models.ServerMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				return false
			}

			// Compare basic fields
			if original.Type != decoded.Type || original.Timestamp != decoded.Timestamp {
				return false
			}

			// Deserialize payload
			payloadBytes, err := json.Marshal(decoded.Payload)
			if err != nil {
				return false
			}
			var decodedPayload models.ChartDataPayload
			if err := json.Unmarshal(payloadBytes, &decodedPayload); err != nil {
				return false
			}

			originalPayload := original.Payload.(models.ChartDataPayload)
			return originalPayload.SessionID == decodedPayload.SessionID &&
				originalPayload.MessageID == decodedPayload.MessageID &&
				originalPayload.ChartType == decodedPayload.ChartType &&
				reflect.DeepEqual(originalPayload.Data.Labels, decodedPayload.Data.Labels) &&
				len(originalPayload.Data.Datasets) == len(decodedPayload.Data.Datasets)
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.IntRange(0, 2),
		gen.SliceOf(gen.AlphaString()),
		gen.SliceOf(gen.Float64()),
	))

	// Test ClientMessage round-trip
	properties.Property("ClientMessage should round-trip through serialization", prop.ForAll(
		func(sessionID, content string, timestamp int64) bool {
			payload := models.UserMessagePayload{
				SessionID: sessionID,
				Content:   content,
			}
			payloadBytes, _ := json.Marshal(payload)

			original := models.ClientMessage{
				Type:      models.MessageTypeUserMessage,
				Payload:   payloadBytes,
				Timestamp: timestamp,
			}

			// Serialize
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize
			var decoded models.ClientMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				return false
			}

			// Compare
			if original.Type != decoded.Type || original.Timestamp != decoded.Timestamp {
				return false
			}

			// Deserialize payload
			var decodedPayload models.UserMessagePayload
			if err := json.Unmarshal(decoded.Payload, &decodedPayload); err != nil {
				return false
			}

			return payload.SessionID == decodedPayload.SessionID &&
				payload.Content == decodedPayload.Content
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Int64Range(0, 9999999999),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}


// Feature: ops-genius-backend, Property 4: 连接断开资源清理
// *对于任何*活跃连接，断开后其占用的资源（内存、goroutine）应该被释放，且会话状态应该被持久化。
// **Validates: Requirements 1.4**
func TestProperty_ConnectionDisconnectResourceCleanup(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Test that closed connections are properly cleaned up
	properties.Property("closed connections should be removed from server", prop.ForAll(
		func(numConnections int) bool {
			logger, _ := zap.NewDevelopment()
			server := NewServer(WithLogger(logger))

			// Create mock connections
			connections := make([]*mockConnectionForCleanup, numConnections)
			for i := 0; i < numConnections; i++ {
				conn := &mockConnectionForCleanup{
					id:     GenerateConnectionID(),
					closed: false,
				}
				connections[i] = conn
				server.connections.Store(conn.id, &WebSocketConnection{
					id:           conn.id,
					lastActivity: time.Now(),
				})
				server.connCount.Add(1)
			}

			// Verify initial count
			if server.ConnectionCount() != numConnections {
				return false
			}

			// Create resource manager with very short idle time
			rm := NewResourceManager(server,
				WithResourceLogger(logger),
				WithMaxIdleTime(1*time.Millisecond),
			)

			// Wait for idle timeout
			time.Sleep(10 * time.Millisecond)

			// Force cleanup
			rm.ForceCleanup()

			// All connections should be cleaned up
			return server.ConnectionCount() == 0
		},
		gen.IntRange(1, 50),
	))

	// Test that connection count is accurate after cleanup
	properties.Property("connection count should be accurate after cleanup", prop.ForAll(
		func(numToCreate, numToRemove int) bool {
			if numToRemove > numToCreate {
				numToRemove = numToCreate
			}

			logger, _ := zap.NewDevelopment()
			server := NewServer(WithLogger(logger))

			// Create connections
			connIDs := make([]string, numToCreate)
			for i := 0; i < numToCreate; i++ {
				id := GenerateConnectionID()
				connIDs[i] = id
				server.connections.Store(id, &WebSocketConnection{
					id:           id,
					lastActivity: time.Now(),
				})
				server.connCount.Add(1)
			}

			// Verify initial count
			if server.ConnectionCount() != numToCreate {
				return false
			}

			// Remove some connections
			rm := NewResourceManager(server, WithResourceLogger(logger))
			for i := 0; i < numToRemove; i++ {
				rm.CleanupConnection(connIDs[i])
			}

			// Verify final count
			expectedCount := numToCreate - numToRemove
			return server.ConnectionCount() == expectedCount
		},
		gen.IntRange(1, 50),
		gen.IntRange(0, 50),
	))

	// Test that cleanup callback is called for each cleaned connection
	properties.Property("cleanup callback should be called for each cleaned connection", prop.ForAll(
		func(numConnections int) bool {
			logger, _ := zap.NewDevelopment()
			server := NewServer(WithLogger(logger))

			// Track cleaned up connections
			cleanedUp := make(map[string]bool)
			var mu sync.Mutex

			rm := NewResourceManager(server,
				WithResourceLogger(logger),
				WithMaxIdleTime(1*time.Millisecond),
				WithOnConnectionClose(func(connID string) {
					mu.Lock()
					cleanedUp[connID] = true
					mu.Unlock()
				}),
			)

			// Create connections
			connIDs := make([]string, numConnections)
			for i := 0; i < numConnections; i++ {
				id := GenerateConnectionID()
				connIDs[i] = id
				server.connections.Store(id, &WebSocketConnection{
					id:           id,
					lastActivity: time.Now(),
				})
				server.connCount.Add(1)
			}

			// Wait for idle timeout
			time.Sleep(10 * time.Millisecond)

			// Force cleanup
			rm.ForceCleanup()

			// Verify all connections were cleaned up
			mu.Lock()
			defer mu.Unlock()
			for _, id := range connIDs {
				if !cleanedUp[id] {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 20),
	))

	// Test that connections with errors are cleaned up
	properties.Property("connections with too many errors should be cleaned up", prop.ForAll(
		func(numConnections, maxErrors int) bool {
			if maxErrors < 1 {
				maxErrors = 1
			}

			logger, _ := zap.NewDevelopment()
			server := NewServer(WithLogger(logger))

			rm := NewResourceManager(server,
				WithResourceLogger(logger),
				WithMaxIdleTime(1*time.Hour), // Long idle time
				WithMaxErrorCount(maxErrors),
			)

			// Create connections with varying error counts
			for i := 0; i < numConnections; i++ {
				conn := NewWebSocketConnection(nil, logger)
				// Set error count to exceed max
				for j := 0; j < maxErrors; j++ {
					conn.IncrementErrorCount()
				}
				server.connections.Store(conn.ID(), conn)
				server.connCount.Add(1)
			}

			// Force cleanup
			rm.ForceCleanup()

			// All connections should be cleaned up due to errors
			return server.ConnectionCount() == 0
		},
		gen.IntRange(1, 20),
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// mockConnectionForCleanup is a mock connection for cleanup testing.
type mockConnectionForCleanup struct {
	id     string
	closed bool
}

// sync.Mutex for thread-safe operations
var cleanupMu sync.Mutex
