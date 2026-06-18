// Package agenthub is Pulse's own agent control plane. Agents hold an OUTBOUND
// WebSocket to Pulse (works through NAT, no inbound/VPN), and Pulse sends
// commands down that channel and awaits replies. This replaces SSH/Tailscale
// for remote agent management with a first-party protocol.
package agenthub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// Command is sent Pulse → agent.
type Command struct {
	ID   string            `json:"id"`
	Cmd  string            `json:"cmd"` // ping | status | install_beyla | remove
	Args map[string]string `json:"args,omitempty"`
}

// Reply is sent agent → Pulse.
type Reply struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

var ErrAgentOffline = errors.New("agent is not connected")

// Hub tracks connected agents per project and routes command replies.
type Hub struct {
	mu      sync.RWMutex
	agents  map[string]map[string]*Client // projectID -> agentID -> client
	pending sync.Map                      // command id -> chan Reply
}

func New() *Hub {
	return &Hub{agents: make(map[string]map[string]*Client)}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.agents[c.projectID] == nil {
		h.agents[c.projectID] = make(map[string]*Client)
	}
	// Replace any stale connection for the same agent id.
	if old := h.agents[c.projectID][c.agentID]; old != nil {
		close(old.send)
	}
	h.agents[c.projectID][c.agentID] = c
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.agents[c.projectID]; ok {
		if set[c.agentID] == c {
			delete(set, c.agentID)
		}
		if len(set) == 0 {
			delete(h.agents, c.projectID)
		}
	}
}

// Connected returns the agent ids currently connected for a project.
func (h *Hub) Connected(projectID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []string
	for id := range h.agents[projectID] {
		out = append(out, id)
	}
	return out
}

func (h *Hub) client(projectID, agentID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.agents[projectID][agentID]
}

// Send delivers a command to an agent and waits for its reply (or ctx timeout).
func (h *Hub) Send(ctx context.Context, projectID, agentID, cmd string, args map[string]string) (Reply, error) {
	c := h.client(projectID, agentID)
	if c == nil {
		return Reply{}, ErrAgentOffline
	}
	id := uuid.NewString()
	replyCh := make(chan Reply, 1)
	h.pending.Store(id, replyCh)
	defer h.pending.Delete(id)

	data, _ := json.Marshal(Command{ID: id, Cmd: cmd, Args: args})
	select {
	case c.send <- data:
	default:
		return Reply{}, ErrAgentOffline
	}

	select {
	case r := <-replyCh:
		return r, nil
	case <-ctx.Done():
		return Reply{}, ctx.Err()
	}
}

func (h *Hub) routeReply(r Reply) {
	if ch, ok := h.pending.Load(r.ID); ok {
		select {
		case ch.(chan Reply) <- r:
		default:
		}
	}
}
