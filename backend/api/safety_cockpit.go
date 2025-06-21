package api

import (
	"encoding/json"
	"log"
	"net/http"
	"qt1-middleware/opik"
	"time"

	"github.com/gorilla/websocket"
)

// SafetyCockpitHandler handles the safety cockpit WebSocket connections
type SafetyCockpitHandler struct {
	upgrader         websocket.Upgrader
	eventManager     *opik.EventStreamManager
	opikClient       *opik.OpikClient
}

// NewSafetyCockpitHandler creates a new safety cockpit handler
func NewSafetyCockpitHandler(eventManager *opik.EventStreamManager, opikClient *opik.OpikClient) *SafetyCockpitHandler {
	return &SafetyCockpitHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		eventManager: eventManager,
		opikClient:   opikClient,
	}
}

// HandleCockpitStream handles WebSocket connections for the safety cockpit
func (h *SafetyCockpitHandler) HandleCockpitStream(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	clientID := generateClientID()
	eventChan := h.eventManager.Subscribe(clientID, 100)
	defer h.eventManager.Unsubscribe(clientID)

	// Send initial data
	h.sendInitialData(conn)

	// Set up channels for communication
	done := make(chan struct{})
	
	// Handle incoming messages
	go h.handleIncomingMessages(conn, clientID, done)

	// Handle outgoing events
	h.handleOutgoingEvents(conn, eventChan, done)
}

// sendInitialData sends historical data to the client
func (h *SafetyCockpitHandler) sendInitialData(conn *websocket.Conn) {
	// Send recent events
	historicalEvents := h.eventManager.GetBufferedEvents(100)
	
	msg := opik.EventWebSocketMessage{
		Type:   "historical",
		Events: historicalEvents,
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send historical data: %v", err)
	}

	// Send aggregated data
	aggregatedData := h.eventManager.GetAggregatedData(5 * time.Minute)
	
	aggMsg := opik.EventWebSocketMessage{
		Type: "aggregated",
		Event: &opik.Event{
			Type: opik.EventTypeMetric,
			Data: aggregatedData,
		},
	}

	if err := conn.WriteJSON(aggMsg); err != nil {
		log.Printf("Failed to send aggregated data: %v", err)
	}
}

// handleIncomingMessages handles messages from the client
func (h *SafetyCockpitHandler) handleIncomingMessages(conn *websocket.Conn, clientID string, done chan struct{}) {
	defer close(done)

	for {
		var msg opik.EventWebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		// Process command
		switch msg.Command {
		case "filter":
			if msg.Filter != nil {
				h.eventManager.SetFilter(clientID, *msg.Filter)
			}
		case "getAggregated":
			// Send aggregated data for different time windows
			windows := []time.Duration{
				1 * time.Minute,
				5 * time.Minute,
				15 * time.Minute,
				1 * time.Hour,
			}
			
			for _, window := range windows {
				data := h.eventManager.GetAggregatedData(window)
				aggMsg := opik.EventWebSocketMessage{
					Type: "aggregated",
					Event: &opik.Event{
						Type: opik.EventTypeMetric,
						Data: map[string]interface{}{
							"window": window.String(),
							"data":   data,
						},
					},
				}
				
				if err := conn.WriteJSON(aggMsg); err != nil {
					log.Printf("Failed to send aggregated data: %v", err)
					return
				}
			}
		case "getMetrics":
			metrics := h.eventManager.GetMetrics()
			metricsMsg := opik.EventWebSocketMessage{
				Type: "metrics",
				Event: &opik.Event{
					Type: opik.EventTypeMetric,
					Data: map[string]interface{}{
						"event_counts": metrics,
					},
				},
			}
			
			if err := conn.WriteJSON(metricsMsg); err != nil {
				log.Printf("Failed to send metrics: %v", err)
				return
			}
		}
	}
}

// handleOutgoingEvents handles streaming events to the client
func (h *SafetyCockpitHandler) handleOutgoingEvents(conn *websocket.Conn, eventChan <-chan opik.Event, done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			
			msg := opik.EventWebSocketMessage{
				Type:  "event",
				Event: &event,
			}
			
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-done:
			return
		}
	}
}

// HandleGetAggregatedData handles HTTP requests for aggregated data
func (h *SafetyCockpitHandler) HandleGetAggregatedData(w http.ResponseWriter, r *http.Request) {
	windowStr := r.URL.Query().Get("window")
	if windowStr == "" {
		windowStr = "5m"
	}

	window, err := time.ParseDuration(windowStr)
	if err != nil {
		http.Error(w, "Invalid window duration", http.StatusBadRequest)
		return
	}

	data := h.eventManager.GetAggregatedData(window)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// HandleGetMetrics handles HTTP requests for event metrics
func (h *SafetyCockpitHandler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.eventManager.GetMetrics()

	response := map[string]interface{}{
		"timestamp":    time.Now(),
		"event_counts": metrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return "client_" + time.Now().Format("20060102150405") + "_" + generateRandomString(8)
}

// generateRandomString generates a random string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}