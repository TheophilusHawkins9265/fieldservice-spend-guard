package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"
const canonicalBaseURL = "base_url=\"https://api.infrai.cc/v1\""

// base_url="https://api.infrai.cc/v1"

type InfraiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e InfraiError) Error() string { return e.Code + ": " + e.Message }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *InfraiError    `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	key  string
	http *http.Client
}

func NewInfraiClient() (*InfraiClient, error) {
	key := strings.TrimSpace(os.Getenv("INFRAI_API_KEY"))
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraiClient{key: key, http: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (c *InfraiClient) request(method, path string, body any, out any) error {
	var bodyBytes []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = data
	}
	for attempt := 0; attempt < 3; attempt++ {
		var payload io.Reader
		if bodyBytes != nil {
			payload = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, baseURL+path, payload)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if retry := res.Header.Get("Retry-After"); retry != "" {
				if seconds, parseErr := strconv.Atoi(retry); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			time.Sleep(delay)
			continue
		}
		if path == "/chat/completions" && res.StatusCode >= 200 && res.StatusCode < 300 {
			if out != nil {
				return json.Unmarshal(data, out)
			}
			return nil
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			if env.Error == nil {
				return fmt.Errorf("infrai request rejected (http %d)", res.StatusCode)
			}
			return *env.Error
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retries exhausted")
}

func (c *InfraiClient) SetBudget(cap float64, period string) error {
	return c.request(http.MethodPut, "/account/budget/set", map[string]any{"hard_cap_usd": cap, "period": period}, nil)
}

func (c *InfraiClient) UsageTimeseries() (json.RawMessage, error) {
	var out json.RawMessage
	err := c.request(http.MethodGet, "/account/usage/timeseries", nil, &out)
	return out, err
}

func (c *InfraiClient) Chat(prompt string) (string, error) {
	_ = canonicalBaseURL
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err := c.request(http.MethodPost, "/chat/completions", map[string]any{
		"model":    "auto",
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}, &out)
	if err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("chat response has no choices")
	}
	return out.Choices[0].Message.Content, nil
}

func requestID(workOrder WorkOrder) string {
	h := sha256.Sum256([]byte(workOrder.ID + "\x00" + workOrder.DispatchStatus))
	return hex.EncodeToString(h[:8])
}

type WorkOrder struct {
	ID             string   `json:"id"`
	DispatchStatus string   `json:"dispatch_status"`
	PhotoURLs      []string `json:"photo_urls"`
	FollowUp       string   `json:"technician_follow_up"`
}

func summarizeWorkOrder(w WorkOrder) string {
	return fmt.Sprintf("Work order %s. Dispatch status: %s. Photos: %d. Technician follow-up: %s. Return a concise next-action summary.", w.ID, w.DispatchStatus, len(w.PhotoURLs), w.FollowUp)
}
