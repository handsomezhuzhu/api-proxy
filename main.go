package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	Version = "1.2.0"
)

var (
	proxyMap = make(map[string]*httputil.ReverseProxy)
	tpl      *template.Template
)

// Denied headers that should not be forwarded to the upstream API
// x-real-ip is denied to prevent leaking internal IP structures if behind multiple proxies
var deniedHeaderPrefixes = []string{"cf-", "forward", "cdn", "x-real-ip"}
var deniedExactHeaders = map[string]bool{
	"host": true,
	// "referer":             true, // 有些 CDN 防盗链可能需要 referer
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailers":            true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

const htmlTemplate = `<!DOCTYPE html>
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
                <tbody>
                    {{range .Items}}
                    <tr>
                        <td><code>{{.Path}}</code></td>
                        <td><span class="target-url">{{.Target}}</span></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        <div class="footer">
            AI API Proxy v{{.Version}} &copy; 2024
            <br>
            Maintainer: <a href="https://zhuzihan.com" target="_blank" style="color: inherit; text-decoration: none; border-bottom: 1px dashed currentColor;">Simon</a>
        </div>
    </div>
</body>
</html>`

type PageData struct {
	Items   []MappingItem
	Version string
}

type MappingItem struct {
	Path   string
	Target string
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

var apiMapping = map[string]string{
	//	"/discord":     "https://discord.com/api",
	//	"/telegram":    "https://api.telegram.org",
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

// 获取请求的真实 IP，优先获取 CDN 传递的 Header
func getClientIP(r *http.Request) string {
	// 阿里 CDN (ESA) 通常会把真实 IP 放在 X-Forwarded-For 的第一个
	// 或者尝试获取 Ali-Cdn-Real-Ip
	if ip := r.Header.Get("Ali-Cdn-Real-Ip"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

func init() {
	// Parse template

	var err error
	tpl, err = template.New("index").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Initialize proxies
	for path, target := range apiMapping {
		targetURL, err := url.Parse(target)
		if err != nil {
			log.Fatalf("Invalid URL for %s: %v", path, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// 1. Optimize connection reuse and streaming
		// FlushInterval is crucial for SSE (Server-Sent Events) to work properly with AI APIs
		proxy.FlushInterval = 100 * time.Millisecond

		// Custom Director to handle headers and request modification
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)

			// Set the Host header to the target host (required by many APIs like OpenAI, Cloudflare)
			req.Host = targetURL.Host

			// Filter denied headers
			// Note: httputil already removes standard hop-by-hop headers
			for k := range req.Header {
				lowerKey := strings.ToLower(k)
				if deniedExactHeaders[lowerKey] {
					req.Header.Del(k)
					continue
				}
				for _, prefix := range deniedHeaderPrefixes {
					if strings.HasPrefix(lowerKey, prefix) {
						req.Header.Del(k)
						break
					}
				}
			}

			// Anonymize forward headers
			req.Header.Del("X-Forwarded-For")
		}

		// 新增：ModifyResponse 强制禁用 CDN 缓存
		proxy.ModifyResponse = func(res *http.Response) error {
			// Nginx 和大部分 CDN 识别这个 Header 来禁用缓冲
			res.Header.Set("X-Accel-Buffering", "no")

			// 确保如果是流式传输，Cache-Control 设置正确
			// content-type 包含 event-stream 时，强制不缓存
			if strings.Contains(res.Header.Get("Content-Type"), "event-stream") {
				res.Header.Set("Cache-Control", "no-cache")
				res.Header.Set("Connection", "keep-alive")
			}
			return nil
		}

		// Optional: Custom error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			clientIP := getClientIP(r)
			log.Printf("[ERROR] Client: %s | Target: %s | Error: %v", clientIP, targetURL.Host, err)

			// 返回 JSON 格式错误，方便 AI 客户端解析
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, `{"error": {"message": "Proxy Connection Error: %v", "type": "proxy_error"}}`, err)
		}

		proxyMap[path] = proxy
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := getClientIP(r)

	// 包装 Writer 记录状态码
	lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// 简单日志
	if r.URL.Path != "/" {
		log.Printf("[REQ] %s %s from %s", r.Method, r.URL.Path, clientIP)
	}

	// 1. Handle Home Page
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		renderHome(lrw)
		return
	}

	// 2. Handle Robots.txt
	if r.URL.Path == "/robots.txt" {
		lrw.Header().Set("Content-Type", "text/plain")
		lrw.WriteHeader(http.StatusOK)
		fmt.Fprint(lrw, "User-agent: *\nDisallow: /")
		return
	}

	// 3. Find Matching Proxy
	// Iterate to find the matching prefix.
	// Optimistically look for exact prefix match or prefix/
	var matchedPrefix string
	path := r.URL.Path

	// Note: Iterating map is random, but since keys are distinct root segments (e.g. /openai),
	// simple prefix check works. For nested paths, one would need to sort keys by length descending.
	for prefix := range apiMapping {
		// Match "/openai" or "/openai/..."
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			matchedPrefix = prefix
			break
		}
	}

	if matchedPrefix == "" {
		http.Error(lrw, "Not Found", http.StatusNotFound)
		log.Printf("[404] Path: %s | IP: %s", path, clientIP)
		return
	}

	// 4. Serve Proxy
	proxy := proxyMap[matchedPrefix]

	// Rewrite the path: remove the prefix
	// Example: /openai/v1/chat/completions -> /v1/chat/completions
	// The SingleHostReverseProxy will append this to the target URL.
	// Target: https://api.openai.com
	// Result: https://api.openai.com/v1/chat/completions
	r.URL.Path = strings.TrimPrefix(path, matchedPrefix)

	// Ensure path starts with / if it became empty
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	// ServeHTTP automatically handles:
	// - Connection reuse (Keep-Alive)
	// - Context cancellation (client disconnects -> stops upstream request)
	// - Header copying (with hop-by-hop removal)
	// - Body copying
	// - Streaming responses
	proxy.ServeHTTP(lrw, r)

	// 只有非 200 或者耗时较长时才打印结束日志，避免日志刷屏
	duration := time.Since(startTime)
	if lrw.statusCode != 200 || duration > 5*time.Second {
		log.Printf("[RES] %d | %v | %s -> %s", lrw.statusCode, duration, path, apiMapping[matchedPrefix])
	}
}

func renderHome(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Prepare data for template
	var items []MappingItem
	for k, v := range apiMapping {
		items = append(items, MappingItem{Path: k, Target: v})
	}

	// Sort by path for consistent display
	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	if err := tpl.Execute(w, PageData{
		Items:   items,
		Version: Version,
	}); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func main() {
	port := "7890"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// Determine the HTTP server settings
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      http.HandlerFunc(handler),
		ReadTimeout:  10 * time.Minute, // Allow long headers/body reading
		WriteTimeout: 0,                // MUST be 0 for streaming responses (SSE) to work indefinitely
		IdleTimeout:  60 * time.Second, // Keep-alive connection idle time
	}

	log.Printf("Starting proxy server on %s (Behind CDN mode)", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
