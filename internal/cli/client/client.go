package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"devflow-backend/internal/cli/config"
)

type Client struct {
	host  string
	token string
	http  *http.Client
}

type PublicClient struct{ Host string }

// New creates a client from the saved config
// If requireAuth is true it returns an error when no token is saved
func New(requiresAuth bool) (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if requiresAuth && cfg.Token == "" {
		return nil, fmt.Errorf("not logged in - run: devflow auth login")
	}
	return &Client{host: cfg.Host, token: cfg.Token, http: &http.Client{}}, nil
}

func (c *Client) url(path string) string {
	return c.host + "/api/v1" + path
}

func (c *Client) do(method, path string, body any) (map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.url(path), bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("bad response from server (status %d)", resp.StatusCode)
	}
	if success, _ := result["success"].(bool); !success {
		if errMsg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("server error (status %d)", resp.StatusCode)
	}
	return result, nil
}

// GET helper — returns the "data" field from the response envelope
func (c *Client) Get(path string) (any, error) {
	res, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return res["data"], nil
}

// POST helper
func (c *Client) Post(path string, body any) (any, error) {
	res, err := c.do("POST", path, body)
	if err != nil {
		return nil, err
	}
	return res["data"], nil
}

// PATCH helper
func (c *Client) Patch(path string, body any) (any, error) {
	res, err := c.do("PATCH", path, body)
	if err != nil {
		return nil, err
	}
	return res["data"], nil
}

// DELETE helper
func (c *Client) Delete(path string) error {
	_, err := c.do("DELETE", path, nil)
	return err
}

// PostMultipart reads a local file, base64-encodes it, and POSTs JSON.
// The upload API expects: { "path": "...", "content": "<base64>", "message": "..." }
func (c *Client) PostMultipart(apiPath, localFilePath, repoFilePath string) error {
	raw, err := os.ReadFile(localFilePath)
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	_, err = c.Post(apiPath, map[string]any{
		"path":    repoFilePath,
		"content": encoded,
		"message": "Upload " + filepath.Base(repoFilePath),
	})
	return err
}
