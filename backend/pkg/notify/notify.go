// Package notify delivers alert notifications to Slack, Telegram or generic webhooks.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
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

// Send delivers the message according to the channel type
// ("slack" | "telegram" | "webhook"). Best-effort: returns an error for
// logging but never panics.
//
// Telegram: set the URL to the bot sendMessage endpoint with the chat id, e.g.
//
//	https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<CHAT_ID>
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
	case "telegram":
		emoji := "🔴"
		if msg.Status == "resolved" {
			emoji = "✅"
		}
		text := fmt.Sprintf("%s %s — %s\n%s", emoji, msg.Title, msg.Status, msg.Description)
		payload := map[string]any{"text": text, "disable_web_page_preview": true}
		// Pull chat_id out of the query so we can POST a clean JSON body.
		if u, perr := neturl.Parse(url); perr == nil {
			q := u.Query()
			if cid := q.Get("chat_id"); cid != "" {
				payload["chat_id"] = cid
				q.Del("chat_id")
				u.RawQuery = q.Encode()
				url = u.String()
			}
		}
		body, err = json.Marshal(payload)
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
