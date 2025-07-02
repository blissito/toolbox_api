package tools

import (
	"io"
	"net/http"
	"strings"
	"time"
)

type WebFetchResult struct {
	Content     string            `json:"content"`
	ContentType string            `json:"content_type"`
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
}

func WebFetch(payload map[string]interface{}) (interface{}, error) {
	url, ok := payload["url"].(string)
	if !ok || url == "" {
		return nil, &ToolError{Message: "URL is required"}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	result := WebFetchResult{
		Content:     string(body),
		ContentType: resp.Header.Get("Content-Type"),
		Status:      resp.StatusCode,
		Headers:     headers,
	}

	return result, nil
}

type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}
