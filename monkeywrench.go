package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mr-b0nzai/MonkeyWrench/internal/runner"
)

func displayHelp() {
	fmt.Println(`MonkeyWrench Help Manual

	Usage: monkeywrench [options]

	Options:
	-mode [string]       The mode to run: 'full' or 'headers' (default: "")
	-file [path]         Path to the file containing URLs (default: "")
	-requests            Print HTTP requests in Burp Suite style (default: false)
	-yaml                Enable YAML output for responses (default: false)
	-H [string]          Comma-separated list of custom headers in 'Key: Value' format. Example: 'X-Forwarded-For: <IP>' (default: "")
	-rate [float]        Rate limit in requests per second (default: 0.0, unlimited)
	-workers [int]       Number of workers to use (default: 10)
	-h                   Display this help manual (default: false)

	Examples:
	# Full mode with custom headers
	monkeywrench -mode=full -file=urls.txt -H="User-Agent: Mozilla, X-Test: Test"

	# Headers mode with stdin input and YAML output
	cat urls.txt | monkeywrench -mode=headers -yaml -requests
	`)
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}

func lineCount(s string) int {
	return len(strings.Split(s, "\n"))
}

// Helper function to parse comma-separated integers
func parseIntList(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}
	var result []int
	for _, v := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

func printInfo(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

func main() {
	// ASCII Art
	asciiArt := ` __  __             _            __        __                   _     
|  \/  | ___  _ __ | | _____ _   \ \      / / __ ___ _ __   ___| |__  
| |\/| |/ _ \| '_ \| |/ / _ \ | | \ \ /\ / / '__/ _ \ '_ \ / __| '_ \ 
| |  | | (_) | | | |   <  __/ |_| |\ V  V /| | |  __/ | | | (__| | | |
|_|  |_|\___/|_| |_|_|\_\___|\__, | \_/\_/ |_|  \___|_| |_|\___|_| |_| 
                             |___/                                    `
	printInfo(asciiArt)
	printInfo("Welcome to the 403 Bypass Tool!")
	printInfo("Author: @Mr_b0nzai")

	// Define flags
	printRequests := flag.Bool("requests", false, "Print HTTP requests in Burp Suite style")
	mode := flag.String("mode", "", "The mode to run: 'full' or 'headers'")
	filePath := flag.String("file", "", "Path to the file containing URLs")
	yamlOutput := flag.Bool("yaml", false, "Enable YAML output for responses")
	customHeaders := flag.String("H", "", "Comma-separated list of custom headers in 'Key: Value' format. Example: 'X-Forwarded-For: <IP>'")
	rateLimit := flag.Float64("rate", 0.0, "Rate limit in requests per second")
	workers := flag.Int("workers", 10, "Number of workers to use")
	method := flag.String("X", "GET", "HTTP method to use")
	help := flag.Bool("h", false, "Display help manual")
	debug := flag.Bool("debug", false, "Enable debug output")
	simple := flag.Bool("simple", false, "Print only the URL")

	// New filter flags
	filterSize := flag.String("fs", "", "Exclude responses by size (comma-separated)")
	filterWords := flag.String("fw", "", "Exclude responses by word count (comma-separated)")
	filterStatus := flag.String("fc", "", "Exclude responses by HTTP status code (comma-separated)")
	filterLines := flag.String("fl", "", "Exclude responses by line count (comma-separated)")
	matchSize := flag.String("ms", "", "Match responses by size (comma-separated)")
	matchWords := flag.String("mw", "", "Match responses by word count (comma-separated)")
	matchStatus := flag.String("mc", "", "Match responses by HTTP status code (comma-separated)")
	matchLines := flag.String("ml", "", "Match responses by line count (comma-separated)")

	flag.Usage = func() {
		displayHelp()
	}

	flag.Parse()

	// Parse all filter and match values
	filterSizeList, err := parseIntList(*filterSize)
	if err != nil {
		fmt.Printf("Invalid filter size value: %v\n", err)
		os.Exit(1)
	}
	filterWordsList, err := parseIntList(*filterWords)
	if err != nil {
		fmt.Printf("Invalid filter words value: %v\n", err)
		os.Exit(1)
	}
	filterStatusList, err := parseIntList(*filterStatus)
	if err != nil {
		fmt.Printf("Invalid filter status value: %v\n", err)
		os.Exit(1)
	}
	filterLinesList, err := parseIntList(*filterLines)
	if err != nil {
		fmt.Printf("Invalid filter lines value: %v\n", err)
		os.Exit(1)
	}

	matchSizeList, err := parseIntList(*matchSize)
	if err != nil {
		fmt.Printf("Invalid match size value: %v\n", err)
		os.Exit(1)
	}
	matchWordsList, err := parseIntList(*matchWords)
	if err != nil {
		fmt.Printf("Invalid match words value: %v\n", err)
		os.Exit(1)
	}
	matchStatusList, err := parseIntList(*matchStatus)
	if err != nil {
		fmt.Printf("Invalid match status value: %v\n", err)
		os.Exit(1)
	}
	matchLinesList, err := parseIntList(*matchLines)
	if err != nil {
		fmt.Printf("Invalid match lines value: %v\n", err)
		os.Exit(1)
	}

	// Validate workers
	if *workers < 1 {
		*workers = 1
	} else if *workers > 100 {
		*workers = 100
	}

	// Display help manual if -h flag is set
	if *help {
		displayHelp()
		os.Exit(0)
	}

	if *mode == "" {
		flag.Usage()
		os.Exit(1)
	}

	var urls []string

	if *filePath == "" {
		// Read URLs from stdin if no file is provided
		urls, err = readLinesFromStdin()
	} else {
		// Read URLs from the provided file
		urls, err = readLines(*filePath)
	}

	if err != nil {
		fmt.Printf("[ERROR] Reading input: %v\n", err)
		os.Exit(1)
	}

	// Parse custom headers
	headers := parseHeaders(*customHeaders)

	// Create runner with rate limit and 10 workers
	r := runner.New(int(*rateLimit), *workers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Dispatch to modes
	switch *mode {
	case "full":
		r.Run(ctx, urls, func(url string) error {
			return fullMode(url, *printRequests, *yamlOutput, headers, filterSizeList, filterWordsList, filterStatusList, filterLinesList, matchSizeList, matchWordsList, matchStatusList, matchLinesList, *method, *debug, *simple)
		})
	case "headers":
		r.Run(ctx, urls, func(url string) error {
			return headersMode(url, *printRequests, *yamlOutput, headers, filterSizeList, filterWordsList, filterStatusList, filterLinesList, matchSizeList, matchWordsList, matchStatusList, matchLinesList, *method, *debug, *simple)
		})
	}
}
