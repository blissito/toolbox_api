package tools

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/microcosm-cc/bluemonday"
)

const (
	maxResponseSize = 5 * 1024 * 1024 // 5MB
	defaultTimeout  = 30 * time.Second
	maxTimeout      = 2 * time.Minute
)

type WebFetchResult struct {
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata"`
}

func WebFetch(payload map[string]interface{}) (interface{}, error) {
	// Parse URL
	url, ok := payload["url"].(string)
	if !ok || url == "" {
		return nil, &ToolError{Message: "URL is required"}
	}

	// Validate URL scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, &ToolError{Message: "URL must start with http:// or https://"}
	}

	// Parse format (default to "html")
	format := "html"
	if f, ok := payload["format"].(string); ok && f != "" {
		switch f {
		case "text", "markdown", "html":
			format = f
		default:
			return nil, &ToolError{Message: "format must be one of: text, markdown, html"}
		}
	}

	// Parse timeout (in seconds)
	timeout := defaultTimeout
	if t, ok := payload["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
		if timeout > maxTimeout {
			timeout = maxTimeout
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 400 {
		return nil, &ToolError{Message: "Request failed with status code: " + strconv.Itoa(resp.StatusCode)}
	}

	// Check content length
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil && size > maxResponseSize {
			return nil, &ToolError{Message: "Response too large (exceeds 5MB limit)"}
		}
	}

	// Read response with size limit
	var buf bytes.Buffer
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	written, err := io.Copy(&buf, limitedReader)
	if err != nil {
		return nil, err
	}
	if written > maxResponseSize {
		return nil, &ToolError{Message: "Response too large (exceeds 5MB limit)"}
	}

	content := buf.Bytes()
	contentType := resp.Header.Get("Content-Type")

	// Process content based on format
	var output string
	switch format {
	case "text":
		if strings.Contains(contentType, "text/html") {
			text, err := extractTextFromHTML(content)
			if err != nil {
				return nil, err
			}
			output = text
		} else {
			output = string(content)
		}

	case "markdown":
		if strings.Contains(contentType, "text/html") {
			markdown, err := convertHTMLToMarkdown(content)
			if err != nil {
				return nil, err
			}
			output = markdown
		} else {
			output = "```\n" + string(content) + "\n```"
		}

	case "html":
		output = string(content)

	default:
		output = string(content)
	}

	// Prepare result
	result := WebFetchResult{
		Output: output,
		Metadata: map[string]string{
			"title": url + " (" + contentType + ")",
		},
	}

	return result, nil
}

func extractTextFromHTML(html []byte) (string, error) {
	// First, sanitize HTML to remove scripts, styles, etc.
	p := bluemonday.StrictPolicy()
	sanitized := p.SanitizeBytes(html)

	// Convert HTML to plain text
	text, err := html2text.FromString(string(sanitized), html2text.Options{
		PrettyTables: true,
	})
	if err != nil {
		return "", err
	}

	// Clean up whitespace
	text = strings.TrimSpace(text)
	return text, nil
}

func convertHTMLToMarkdown(html []byte) (string, error) {
	// First, sanitize HTML
	p := bluemonday.StrictPolicy()
	sanitized := p.SanitizeBytes(html)

	// Convert HTML to plain text first to handle complex HTML better
	text, err := html2text.FromString(string(sanitized), html2text.Options{
		PrettyTables: true,
	})
	if err != nil {
		return "", err
	}

	return text, nil
}

type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}
