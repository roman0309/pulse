// Package notify delivers alert notifications to Slack or generic webhooks.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Notifier struct{ client *http.Client }

func New() *Notifier {
	return &Notifier{client: &http.Client{Timeout: 5 * time.Second}}
}

// Message is the alert payload delivered to a channel.
type Message struct {
	Title       string  `json:"title"`
	Service     string  `json:"service"`
	Metric      string  `json:"metric"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"`
	Status      string  `json:"status"` // firing | resolved
	Description string  `json:"description"`
}

// Send delivers the message according to the channel type ("slack" | "webhook").
// Best-effort: returns an error for logging but never panics.
func (n *Notifier) Send(ctx context.Context, channelType, url string, msg Message) error {
	if url == "" || channelType == "" || channelType == "none" {
		return nil
	}
	var body []byte
	var err error
	switch channelType {
	case "slack":
		emoji := "🔴"
		if msg.Status == "resolved" {
			emoji = "✅"
		}
		text := fmt.Sprintf("%s *%s* — %s\n%s", emoji, msg.Title, msg.Status, msg.Description)
		body, err = json.Marshal(map[string]string{"text": text})
	default: // generic webhook
		body, err = json.Marshal(map[string]any{"alert": msg})
	}
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("notify %s returned %s", channelType, resp.Status)
	}
	return nil
}
