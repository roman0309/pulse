package ws

import (
	"encoding/json"
	"sync"
)

// Event is a realtime message broadcast to subscribers of a project channel.
type Event struct {
	Type      string      `json:"type"` // alert | metric | timeline
	ProjectID string      `json:"project_id"`
	Payload   interface{} `json:"payload"`
}

// Hub maintains active WebSocket clients grouped by project and fans out events.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // projectID -> set of clients
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]bool)}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.projectID] == nil {
		h.clients[c.projectID] = make(map[*Client]bool)
	}
	h.clients[c.projectID][c] = true
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.projectID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.projectID)
		}
	}
}

// Broadcast sends an event to every client subscribed to the event's project.
func (h *Hub) Broadcast(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[ev.ProjectID] {
		select {
		case c.send <- data:
		default:
			// drop slow consumers
		}
	}
}
