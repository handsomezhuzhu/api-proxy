package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

var apiMapping = map[string]string{
	"/discord":     "https://discord.com/api",
	"/telegram":    "https://api.telegram.org",
	"/openai":      "https://api.openai.com",
	"/claude":      "https://api.anthropic.com",
	"/gemini":      "https://generativelanguage.googleapis.com",
	"/meta":        "https://www.meta.ai/api",
	"/groq":        "https://api.groq.com/openai",
	"/xai":         "https://api.x.ai",
	"/cohere":      "https://api.cohere.ai",
	"/huggingface": "https://api-inference.huggingface.co",
	"/together":    "https://api.together.xyz",
	"/novita":      "https://api.novita.ai",
	"/portkey":     "https://api.portkey.ai",
	"/fireworks":   "https://api.fireworks.ai",
	"/openrouter":  "https://openrouter.ai/api",
	"/cerebras":    "https://api.cerebras.ai",
}

var deniedHeaders = []string{"host", "referer", "cf-", "forward", "cdn"}

func isAllowedHeader(key string) bool {
	for _, deniedHeader := range deniedHeaders {
		if strings.Contains(strings.ToLower(key), deniedHeader) {
			return false
		}
	}
	return true
}

func targetURL(pathname string) string {
	split := strings.Index(pathname[1:], "/")
	prefix := pathname[:split+1]
	if base, exists := apiMapping[prefix]; exists {
		return base + pathname[len(prefix):]
	}
	return ""
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		var paths []string
		for k := range apiMapping {
			paths = append(paths, k)
		}
		sort.Strings(paths)

		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI API Proxy</title>
    <style>
        :root {
            --primary-color: #4a90e2;
            --bg-color: #f4f6f9;
            --card-bg: #ffffff;
            --text-color: #333333;
            --text-secondary: #666666;
            --border-color: #eaeaea;
            --success-bg: #d4edda;
            --success-text: #155724;
            --code-bg: #f8f9fa;
            --code-text: #e83e8c;
            --hover-bg: #f8f9fa;
        }
        body {
            font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Roboto, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            margin: 0;
            padding: 40px 20px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: var(--card-bg);
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            padding: 40px;
        }
        h1 {
            margin-top: 0;
            margin-bottom: 20px;
            font-size: 28px;
            font-weight: 700;
            color: #2d3748;
            border-bottom: 2px solid var(--border-color);
            padding-bottom: 20px;
        }
        h2 {
            font-size: 20px;
            margin-top: 30px;
            margin-bottom: 15px;
            color: #4a5568;
        }
        .status {
            background-color: var(--success-bg);
            color: var(--success-text);
            padding: 12px 20px;
            border-radius: 8px;
            display: inline-flex;
            align-items: center;
            font-weight: 500;
            margin-bottom: 24px;
        }
        .status-icon {
            margin-right: 8px;
            font-size: 1.2em;
        }
        p {
            color: var(--text-secondary);
            margin-bottom: 24px;
        }
        .table-container {
            border-radius: 8px;
            border: 1px solid var(--border-color);
            overflow: hidden;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            background: white;
        }
        th, td {
            text-align: left;
            padding: 16px;
            border-bottom: 1px solid var(--border-color);
        }
        th {
            background-color: #f7fafc;
            font-weight: 600;
            color: #4a5568;
            text-transform: uppercase;
            font-size: 0.85rem;
            letter-spacing: 0.05em;
        }
        tr:last-child td {
            border-bottom: none;
        }
        tr:hover {
            background-color: var(--hover-bg);
        }
        code {
            background-color: var(--code-bg);
            color: var(--code-text);
            padding: 4px 8px;
            border-radius: 4px;
            font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
            font-size: 0.9em;
            border: 1px solid #e2e8f0;
        }
        .target-url {
            color: var(--text-secondary);
            font-family: monospace;
            font-size: 0.95em;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 0.9em;
            color: #a0aec0;
        }
        @media (max-width: 640px) {
            body { padding: 20px 15px; }
            .container { padding: 25px; }
            th, td { padding: 12px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>AI API Proxy Service</h1>
        <p style="margin-top: -15px; margin-bottom: 25px; color: var(--text-secondary);">
            Maintained by <a href="https://zhuzihan.com" target="_blank" style="color: var(--primary-color); text-decoration: none; font-weight: 500;">Simon</a>
        </p>
        <div class="status">
            <span class="status-icon">✅</span>
            Service is active and running
        </div>
        <p>This service routes requests to various AI provider APIs through a unified interface.</p>
        
        <h2>Available Endpoints</h2>
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th width="30%">Path Prefix</th>
                        <th>Target Service URL</th>
                    </tr>
                </thead>
                <tbody>`
		for _, path := range paths {
			target := apiMapping[path]
			html += fmt.Sprintf("<tr><td><code>%s</code></td><td><span class=\"target-url\">%s</span></td></tr>", path, target)
		}
		html += `</tbody></table></div>
        <div class="footer">
            AI API Proxy &copy; 2024
            <br>
            Maintainer: <a href="https://zhuzihan.com" target="_blank" style="color: inherit; text-decoration: none; border-bottom: 1px dashed currentColor;">Simon</a>
        </div>
    </div>
</body>
</html>`
		fmt.Fprint(w, html)
		return
	}

	if r.URL.Path == "/robots.txt" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "User-agent: *\nDisallow: /")
		return
	}

	query := r.URL.RawQuery

	if query != "" {
		query = "?" + query
	}

	targetURL := targetURL(r.URL.Path + query)

	if targetURL == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Create new request
	client := &http.Client{}
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		if isAllowedHeader(key) {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}
	}

	// Make the request
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Failed to fetch: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response: %v", err)
	}
}

func main() {
	port := "7890"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	http.HandleFunc("/", handler)
	log.Printf("Starting server on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
