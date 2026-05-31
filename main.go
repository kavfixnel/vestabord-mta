package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const apiURL = "https://cloud.vestaboard.com/"

type sendRequest struct {
	Text   string `json:"text"`
	Forced bool   `json:"forced,omitempty"`
}

type sendResponse struct {
	Status  string `json:"status"`
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Error   string `json:"error"`
}

func loadToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		if strings.TrimSpace(key) == "VESTABORD_TOKEN" {
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			if value == "" {
				return "", fmt.Errorf("VESTABORD_TOKEN is empty in %s", path)
			}
			return value, nil
		}
	}

	return "", fmt.Errorf("VESTABORD_TOKEN not found in %s", path)
}

func sendMessage(token, text string, forced bool) (*sendResponse, error) {
	body, err := json.Marshal(sendRequest{Text: text, Forced: forced})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vestaboard-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result sendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != "" {
			return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, result.Error)
		}
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, string(respBody))
	}

	return &result, nil
}

func main() {
	envPath := flag.String("env", ".env", "path to .env file")
	forced := flag.Bool("forced", false, "send even during quiet hours")
	flag.Parse()

	text := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if text == "" {
		fmt.Fprintln(os.Stderr, "usage: vestaboard [-env .env] [-forced] <message>")
		os.Exit(1)
	}

	token, err := loadToken(*envPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result, err := sendMessage(token, text, *forced)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("sent message (id: %s)\n", result.ID)
}
