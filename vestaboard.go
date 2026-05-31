package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const vestaboardAPIURL = "https://cloud.vestaboard.com/"

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

func sendMessage(token, text string, forced bool) (*sendResponse, error) {
	body, err := json.Marshal(sendRequest{Text: text, Forced: forced})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, vestaboardAPIURL, bytes.NewReader(body))
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
