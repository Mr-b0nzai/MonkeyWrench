package main

import (
	"flag"
	"fmt"
	"os"
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
	customHeaders := flag.String("headers", "", "Comma-separated list of custom headers in 'Key: Value' format. Example: 'X-Forwarded-For: <IP>'")

	flag.Usage = func() {
		fmt.Println("Usage: go run main.go -mode=<mode> -file=<url_file> [-requests]")
		flag.PrintDefaults()
	}
	flag.Parse()

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

	// Dispatch to modes
	switch *mode {
	case "full":
		for _, url := range urls {
			if url == "" || url == "\x00" { // Skip empty lines
				continue
			}
			fullMode(url, *printRequests, *yamlOutput, headers) // Pass the printRequests flag
		}
	case "headers":
		for _, url := range urls {
			if url == "" || url == "\x00" { // Skip empty lines
				continue
			}
			headersMode(url, *printRequests, *yamlOutput, headers) // Pass the printRequests flag
		}
	default:
		flag.Usage()
	}
}
