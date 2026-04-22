package analyzer

// Core logic: fetch, parse, analyze
// Package analyzer provides HTML page analysis functionality.
// It fetches a URL and extracts metadata including HTML version, page title,
// heading counts, link classification, and login form detection.

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Result holds all extracted information from a single page analysis.
type Result struct {
	URL               string         `json:"url"`
	HTMLVersion       string         `json:"html_version"`
	Title             string         `json:"title"`
	Headings          map[string]int `json:"headings"`
	InternalLinks     int            `json:"internal_links"`
	ExternalLinks     int            `json:"external_links"`
	InaccessibleLinks int            `json:"inaccessible_links"`
	HasLoginForm      bool           `json:"has_login_form"`
}

// AnalyzeError wraps errors that include an HTTP status code.
type AnalyzeError struct {
	StatusCode int
	Message    string
}

func (e *AnalyzeError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// Analyze fetches the page at rawURL and returns a populated Result.
func Analyze(rawURL string) (*Result, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, &AnalyzeError{StatusCode: 400, Message: "invalid URL format"}
	}

	resp, err := httpClient.Get(parsed.String())
	if err != nil {
		return nil, &AnalyzeError{StatusCode: 0, Message: fmt.Sprintf("could not reach URL: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &AnalyzeError{
			StatusCode: resp.StatusCode,
			Message:    httpStatusDescription(resp.StatusCode),
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10 MB cap
	if err != nil {
		return nil, &AnalyzeError{StatusCode: 500, Message: "failed to read response body"}
	}

	result := &Result{
		URL:      parsed.String(),
		Headings: make(map[string]int),
	}

	result.HTMLVersion = detectHTMLVersion(string(body))

	links, err := parseHTML(string(body), parsed, result)
	if err != nil {
		return nil, &AnalyzeError{StatusCode: 500, Message: "failed to parse HTML"}
	}

	result.InaccessibleLinks = countInaccessibleLinks(links)

	return result, nil
}

// detectHTMLVersion inspects the DOCTYPE declaration to determine the HTML version.
func detectHTMLVersion(body string) string {
	upper := strings.ToUpper(body)
	idx := strings.Index(upper, "<!DOCTYPE")
	if idx == -1 {
		return "Unknown"
	}

	doctype := upper[idx:]
	end := strings.Index(doctype, ">")
	if end != -1 {
		doctype = doctype[:end+1]
	}

	switch {
	case strings.Contains(doctype, "HTML 4.01") && strings.Contains(doctype, "STRICT"):
		return "HTML 4.01 Strict"
	case strings.Contains(doctype, "HTML 4.01") && strings.Contains(doctype, "TRANSITIONAL"):
		return "HTML 4.01 Transitional"
	case strings.Contains(doctype, "HTML 4.01") && strings.Contains(doctype, "FRAMESET"):
		return "HTML 4.01 Frameset"
	case strings.Contains(doctype, "HTML 4.01"):
		return "HTML 4.01"
	case strings.Contains(doctype, "XHTML 1.0") && strings.Contains(doctype, "STRICT"):
		return "XHTML 1.0 Strict"
	case strings.Contains(doctype, "XHTML 1.0") && strings.Contains(doctype, "TRANSITIONAL"):
		return "XHTML 1.0 Transitional"
	case strings.Contains(doctype, "XHTML 1.0") && strings.Contains(doctype, "FRAMESET"):
		return "XHTML 1.0 Frameset"
	case strings.Contains(doctype, "XHTML 1.1"):
		return "XHTML 1.1"
	case strings.Contains(doctype, "HTML 3.2"):
		return "HTML 3.2"
	case strings.Contains(doctype, "HTML 2.0"):
		return "HTML 2.0"
	case strings.Contains(doctype, "<!DOCTYPE HTML>"):
		return "HTML5"
	default:
		// Fallback to HTML5 for any other DOCTYPE or if unspecified (but idx != -1)
		return "HTML5"
	}
}

// parseHTML walks the HTML token tree extracting title, headings, links, and login form presence.
// It returns all href values found for subsequent link checking.
func parseHTML(body string, base *url.URL, result *Result) ([]string, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(body))

	var links []string
	inTitle := false
	var titleBuilder strings.Builder

	// Track login form detection state
	inForm := false
	hasPasswordField := false
	hasSubmitField := false

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		token := tokenizer.Token()

		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tag := strings.ToLower(token.Data)

			switch tag {
			case "title":
				inTitle = true

			case "h1", "h2", "h3", "h4", "h5", "h6":
				result.Headings[tag]++

			case "form":
				inForm = true
				hasPasswordField = false
				hasSubmitField = false

			case "input":
				if !inForm {
					break
				}
				inputType := getAttr(token, "type")
				switch strings.ToLower(inputType) {
				case "password":
					hasPasswordField = true
				case "submit":
					hasSubmitField = true
				}

			case "button":
				if inForm {
					btnType := strings.ToLower(getAttr(token, "type"))
					if btnType == "submit" || btnType == "" {
						hasSubmitField = true
					}
				}

			case "a":
				href := getAttr(token, "href")
				if href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
					break
				}

				resolved := resolveURL(base, href)
				if resolved == "" {
					break
				}

				// Only add http/https links to the check list
				if strings.HasPrefix(resolved, "http://") || strings.HasPrefix(resolved, "https://") {
					links = append(links, resolved)
				}

				if isSameHost(base, resolved) {
					result.InternalLinks++
				} else {
					result.ExternalLinks++
				}
			}

		case html.EndTagToken:
			tag := strings.ToLower(token.Data)
			switch tag {
			case "title":
				inTitle = false
				result.Title = strings.TrimSpace(titleBuilder.String())
			case "form":
				if inForm && hasPasswordField && hasSubmitField {
					result.HasLoginForm = true
				}
				inForm = false
			}

		case html.TextToken:
			if inTitle {
				titleBuilder.WriteString(token.Data)
			}
		}
	}

	return links, nil
}

// countInaccessibleLinks sends HEAD requests concurrently to all links and
// counts those that return a non-2xx/3xx status or fail to connect.
func countInaccessibleLinks(links []string) int {
	// Deduplicate to avoid hammering the same URL.
	seen := make(map[string]bool, len(links))
	unique := links[:0]
	for _, l := range links {
		if !seen[l] {
			seen[l] = true
			unique = append(unique, l)
		}
	}

	type result struct{ inaccessible bool }
	ch := make(chan result, len(unique))

	// Cap concurrency to avoid overwhelming the network.
	sem := make(chan struct{}, 20)

	checkClient := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // follow redirects
		},
	}

	for _, link := range unique {
		link := link
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			resp, err := checkClient.Head(link)
			if err != nil {
				ch <- result{inaccessible: true}
				return
			}
			resp.Body.Close()
			ch <- result{inaccessible: resp.StatusCode >= 400}
		}()
	}

	count := 0
	for range unique {
		r := <-ch
		if r.inaccessible {
			count++
		}
	}
	return count
}

// resolveURL resolves href relative to the page's base URL.
func resolveURL(base *url.URL, href string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(ref)
	return resolved.String()
}

// isSameHost returns true when the resolved URL shares the same hostname as the base.
func isSameHost(base *url.URL, resolved string) bool {
	u, err := url.Parse(resolved)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Hostname(), base.Hostname())
}

// getAttr retrieves the value of a named attribute from a token.
func getAttr(t html.Token, name string) string {
	for _, a := range t.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

// httpStatusDescription returns a brief human-readable explanation for common HTTP codes.
func httpStatusDescription(code int) string {
	switch code {
	case 301, 302, 303, 307, 308:
		return "redirect response (too many redirects)"
	case 400:
		return "bad request"
	case 401:
		return "authentication required"
	case 403:
		return "access forbidden"
	case 404:
		return "page not found"
	case 405:
		return "method not allowed"
	case 410:
		return "page permanently removed"
	case 429:
		return "too many requests – rate limited"
	case 500:
		return "internal server error"
	case 502:
		return "bad gateway"
	case 503:
		return "service unavailable"
	case 504:
		return "gateway timeout"
	default:
		return http.StatusText(code)
	}
}
