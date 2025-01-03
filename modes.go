package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// decodeUTF16 decodes a UTF-16 byte slice to a UTF-8 string
func decodeUTF16(b []byte) string {
	if len(b)%2 != 0 {
		return ""
	}
	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		u16s[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	return string(utf16.Decode(u16s))
}

func removeBOM(line []byte) []byte {
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM := []byte{0xFF, 0xFE}
	utf16BEBOM := []byte{0xFE, 0xFF}

	// Remove BOM if present
	if bytes.HasPrefix(line, utf8BOM) {
		return line[len(utf8BOM):] // Remove UTF-8 BOM
	} else if bytes.HasPrefix(line, utf16LEBOM) || bytes.HasPrefix(line, utf16BEBOM) {
		return []byte(decodeUTF16(line)) // Decode UTF-16 to UTF-8
	}

	// Clean any non-printable characters (like null bytes) from the URL
	return bytes.Map(func(r rune) rune {
		if r == 0 { // Remove null bytes
			return -1
		}
		return r
	}, line)
}

// normalizeURL ensures the URL has a scheme (http or https)
func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL) // Clean leading/trailing spaces
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}
	// Remove BOM and unwanted control characters
	rawURL = string(removeBOM([]byte(rawURL)))

	// Check if the URL already has a scheme (http:// or https://)
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Remove \r and \n
	rawURL = strings.ReplaceAll(rawURL, "\r", "")
	rawURL = strings.ReplaceAll(rawURL, "\n", "")

	// Parse the URL to ensure it's valid
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	// Return the cleaned and validated URL
	return parsedURL.String(), nil
}

// readLines reads a file and returns a slice of strings containing each line (URL)
func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	isFirstLine := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if isFirstLine {
			line = removeBOM(line) // Handle BOM on the first line
			isFirstLine = false
		}

		if !utf8.Valid(line) {
			line = []byte(decodeUTF16(line)) // Decode UTF-16 to UTF-8
		}

		trimmedLine := strings.TrimSpace(string(line))
		if trimmedLine != "" {
			lines = append(lines, trimmedLine)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// Struct to hold the response details for YAML formatting
type ResponseDetails struct {
	StatusCode int    `yaml:"status"`
	Body       string `yaml:"body"`
}

// full mode: sends a request to each URL in the file with all settings enabled
func fullMode(url string, printRequests bool, yamlOutput bool, customHeaders map[string]string) {
	headersMode(url, printRequests, yamlOutput, customHeaders)
}

// headersMode function
func headersMode(url string, printRequests bool, yamlOutput bool, customHeaders map[string]string) {
	client := &http.Client{Timeout: 15 * time.Second}
	url, err := normalizeURL(url)
	if err != nil {
		fmt.Printf("[ERROR] Normalizing URL: %v\n", err)
		return
	}

	// List of headers to add
	headers := map[string]string{
		"X-Forwarded-For":           "127.0.0.1",
		"Client-IP":                 "127.0.0.1",
		"Cluster-Client-IP":         "127.0.0.1",
		"Connection":                "keep-alive",
		"Content-Length":            "0",
		"Forwarded-For":             "127.0.0.1",
		"Host":                      "example.com",
		"Referer":                   "https://example.com",
		"True-Client-IP":            "127.0.0.1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36",
		"X-Custom-IP-Authorization": "127.0.0.1",
		"X-Forwarded":               "127.0.0.1",
		"X-Forwarded-Port":          "443",
		"X-Original-URL":            "/original-url",
		"X-Originating-IP":          "127.0.0.1",
		"X-ProxyUser-Ip":            "127.0.0.1",
		"X-Remote-Addr":             "127.0.0.1",
		"X-Remote-IP":               "127.0.0.1",
		"X-Rewrite-URL":             "/rewrite-url",
	}

	for key, value := range headers {
		// Create a new request for each header
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("[ERROR] Creating request for %s: %v\n", url, err)
			continue
		}

		// Add the current header
		req.Header.Add(key, value)
		for k, v := range customHeaders {
			req.Header.Add(k, v)
		}

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[ERROR] Request to %s failed: %v\n", url, err)
			continue
		}
		defer resp.Body.Close()

		// Read the response body and handle potential errors
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[ERROR] Reading response body failed: %v\n", err)
			continue
		}

		// Create a struct with response details
		responseDetails := ResponseDetails{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}

		// Conditionally print the response in YAML format
		if yamlOutput {
			yamlData, err := yaml.Marshal(&responseDetails)
			if err != nil {
				fmt.Printf("[ERROR] Marshaling response to YAML failed: %v\n", err)
				continue
			}
			// Print the YAML output
			fmt.Println("Response in YAML format:")
			fmt.Println(string(yamlData))
		}

		// Print the request method, URL, and headers for debugging
		fmt.Printf("[%s] URL: %s | Status: %d | Header: %s=%s\n", resp.Request.Method, url, resp.StatusCode, key, value)

		// Conditionally print the HTTP request (debugging)
		if printRequests {
			fmt.Println("[DEBUG] Printing request in Burp Suite format...")
			printBurpStyleRequest(req) // This should now print the request in Burp format
		}
	}
}

// printBurpStyleRequest prints an HTTP request in Burp Suite-style format
func printBurpStyleRequest(req *http.Request) {
	// Start with the request line
	fmt.Printf("%s %s HTTP/1.1\n", req.Method, req.URL.RequestURI())
	fmt.Printf("Host: %s\n", req.URL.Host)

	// Ensure all headers are printed, including defaults
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "403-Bypass-Tool")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

	// Add headers
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	// Add a blank line before the body (if any)
	fmt.Println()
}

func readLinesFromStdin() ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func parseHeaders(headerString string) map[string]string {
	headers := make(map[string]string)
	if headerString == "" {
		return headers
	}

	headerPairs := strings.Split(headerString, ",")
	for _, pair := range headerPairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		} else {
			fmt.Printf("[WARNING] Invalid header format: %s\n", pair)
		}
	}

	return headers
}
