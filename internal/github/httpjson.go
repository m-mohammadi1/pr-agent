package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func postJSON(ctx context.Context, client *http.Client, url, token string, body, dest any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshalJSON(data []byte, dest any) error {
	return json.Unmarshal(data, dest)
}
