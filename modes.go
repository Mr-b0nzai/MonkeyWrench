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

// Add color constants at top of file
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// Add helper function for status code coloring
func getStatusColor(code int) string {
	switch {
	case code >= 500:
		return colorRed
	case code >= 400:
		return colorYellow
	case code >= 300:
		return colorBlue
	case code >= 200:
		return colorGreen
	default:
		return colorReset
	}
}

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

// printBurpStyleRequest prints an HTTP request in Burp Suite-style format
// contains checks if a slice contains a specific value
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

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

// fullMode now takes a single URL and returns an error
func fullMode(url string, printRequests bool, yamlOutput bool, customHeaders map[string]string, filterSize []int, filterWords []int, filterStatus []int, filterLines []int, matchSize []int, matchWords []int, matchStatus []int, matchLines []int, method string, debug bool, simple bool) error {
	return headersMode(url, printRequests, yamlOutput, customHeaders, filterSize, filterWords, filterStatus, filterLines, matchSize, matchWords, matchStatus, matchLines, method, debug, simple)
}

// headersMode function
func headersMode(url string, printRequests bool, yamlOutput bool, customHeaders map[string]string, filterSize []int, filterWords []int, filterStatus []int, filterLines []int, matchSize []int, matchWords []int, matchStatus []int, matchLines []int, method string, debug bool, simple bool) error {
	// Ensure method is uppercase
	method = strings.ToUpper(method)

	// Add debug logging for method
	if debug {
		fmt.Printf("[DEBUG] Using HTTP method: %s\n", method)
	}
	if url == "" || url == "\x00" {
		return nil
	}

	// Create custom client with CheckRedirect function
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Preserve the original method on redirect
			req.Method = method
			return nil
		},
	}
	url, err := normalizeURL(url)
	if err != nil {
		return fmt.Errorf("normalizing URL: %v", err)
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
		if debug {
			fmt.Printf("[DEBUG] Creating request with method %s for header %s=%s\n", method, key, value)
		}

		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return fmt.Errorf("creating request: %v", err)
		}

		// Verify request method before adding headers
		if req.Method != method {
			return fmt.Errorf("method mismatch before headers: expected %s, got %s", method, req.Method)
		}

		req.Header.Add(key, value)
		for k, v := range customHeaders {
			req.Header.Add(k, v)
		}

		// Verify request method before sending
		if req.Method != method {
			return fmt.Errorf("method mismatch before send: expected %s, got %s", method, req.Method)
		}

		if debug {
			fmt.Printf("[DEBUG] Sending request with method: %s\n", req.Method)
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Verify response request method
		if resp.Request.Method != method {
			fmt.Printf("[ERROR] Method changed during request: expected %s, got %s\n", method, resp.Request.Method)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %v", err)
		}

		// Filter responses
		if len(filterSize) > 0 && contains(filterSize, len(body)) {
			continue
		}
		if len(filterWords) > 0 && contains(filterWords, wordCount(string(body))) {
			continue
		}
		if len(filterStatus) > 0 && contains(filterStatus, resp.StatusCode) {
			continue
		}
		if len(filterLines) > 0 && contains(filterLines, lineCount(string(body))) {
			continue
		}

		// Match responses
		if len(matchSize) > 0 && !contains(matchSize, len(body)) {
			continue
		}
		if len(matchWords) > 0 && !contains(matchWords, wordCount(string(body))) {
			continue
		}
		if len(matchStatus) > 0 && !contains(matchStatus, resp.StatusCode) {
			continue
		}
		if len(matchLines) > 0 && !contains(matchLines, lineCount(string(body))) {
			continue
		}

		responseDetails := ResponseDetails{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}

		if yamlOutput {
			yamlData, err := yaml.Marshal(&responseDetails)
			if err != nil {
				return fmt.Errorf("marshaling YAML: %v", err)
			}
			fmt.Println("Response in YAML format:")
			fmt.Println(string(yamlData))
		}

		if simple {
			fmt.Fprintln(os.Stdout, url)
		} else {
			// Print detailed output to stdout
			fmt.Fprintf(os.Stdout, "%s%s | %d | %s | %s: %s | Size: %d%s\n",
				getStatusColor(resp.StatusCode),
				resp.Request.Method,
				resp.StatusCode,
				url,
				key,
				value,
				len(body),
				colorReset)
		}

		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Request completed: %s\n", url)
		}

		if printRequests {
			fmt.Println("[DEBUG] Printing request in Burp Suite format...")
			printBurpStyleRequest(req)
		}

	}

	return nil
}
