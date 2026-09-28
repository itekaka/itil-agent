package transport

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	AgentID    string
	Secret     string
	HTTPClient *http.Client
}

func NewClient(baseURL, agentID, secret string, skipTLS bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		AgentID: agentID,
		Secret:  secret,
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: tr,
		},
	}
}

func (c *Client) PostMethod(method string, body any) (map[string]any, int, error) {
	url := c.BaseURL + "/api/method/" + method
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", c.AgentID)
	req.Header.Set("X-Agent-Secret", c.Secret)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("invalid JSON (%d): %s", resp.StatusCode, string(data))
	}
	return parsed, resp.StatusCode, nil
}

