package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"monkeywrench/internal/runner"
)

func main() {
	// ASCII Art
	asciiArt := ` __  __             _            __        __                   _     
|  \/  | ___  _ __ | | _____ _   \ \      / / __ ___ _ __   ___| |__  
| |\/| |/ _ \| '_ \| |/ / _ \ | | \ \ /\ / / '__/ _ \ '_ \ / __| '_ \ 
| |  | | (_) | | | |   <  __/ |_| |\ V  V /| | |  __/ | | | (__| | | |
|_|  |_|\___/|_| |_|_|\_\___|\__, | \_/\_/ |_|  \___|_| |_|\___|_| |_| 
                             |___/                                    `
	fmt.Println(asciiArt)
	fmt.Println("Welcome to the 403 Bypass Tool!")
	fmt.Println("Author: @Mr_b0nzai")

	// Define flags
	printRequests := flag.Bool("requests", false, "Print HTTP requests in Burp Suite style")
	mode := flag.String("mode", "", "The mode to run: 'full' or 'headers'")
	filePath := flag.String("file", "", "Path to the file containing URLs")
	yamlOutput := flag.Bool("yaml", false, "Enable YAML output for responses")
	customHeaders := flag.String("H", "", "Comma-separated list of custom headers in 'Key: Value' format. Example: 'X-Forwarded-For: <IP>'")
	rateLimit := flag.Float64("rate", 0.0, "Rate limit in requests per second")
	workers := flag.Int("workers", 10, "Number of workers to use")

	flag.Usage = func() {
		fmt.Println("Usage: go run main.go -mode=<mode> -file=<url_file> [-requests]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Validate workers
	if *workers < 1 {
		*workers = 1
	} else if *workers > 100 {
		*workers = 100
	}

	if *mode == "" {
		flag.Usage()
		os.Exit(1)
	}

	var urls []string
	var err error

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

	// Create a channel to count requests
	requestCount := make(chan int, 1)

	// Launch a goroutine to print the request rate
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		var count int
		for {
			select {
			case <-ticker.C:
				fmt.Printf("Requests per second: %d\n", count)
				count = 0
			case n := <-requestCount:
				count += n
			case <-ctx.Done():
				return
			}
		}
	}()

	// Dispatch to modes
	switch *mode {
	case "full":
		r.Run(ctx, urls, func(url string) error {
			return fullMode(url, *printRequests, *yamlOutput, headers, requestCount)
		})
	case "headers":
		r.Run(ctx, urls, func(url string) error {
			return headersMode(url, *printRequests, *yamlOutput, headers, requestCount)
		})
	default:
		flag.Usage()
	}
}
