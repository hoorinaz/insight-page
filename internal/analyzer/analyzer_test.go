package analyzer

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDetectHTMLVersion(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{"HTML5", `<!DOCTYPE html><html>`, "HTML5"},
		{"HTML5 uppercase", `<!DOCTYPE HTML><html>`, "HTML5"},
		{"HTML 4.01 Strict", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`, "HTML 4.01 Strict"},
		{"HTML 4.01 Transitional", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">`, "HTML 4.01 Transitional"},
		{"XHTML 1.0 Strict", `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN">`, "XHTML 1.0 Strict"},
		{"No doctype", `<html><body>hello</body></html>`, "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectHTMLVersion(tc.body)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestParseHTMLHeadings(t *testing.T) {
	body := `<!DOCTYPE html><html><head><title>Test</title></head><body>
		<h1>Main</h1>
		<h2>Sub</h2><h2>Sub2</h2>
		<h3>Sub-sub</h3>
	</body></html>`

	base, _ := url.Parse("https://example.com")
	result := &Result{Headings: make(map[string]int)}
	parseHTML(body, base, result)

	if result.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", result.Title)
	}
	if result.Headings["h1"] != 1 {
		t.Errorf("expected 1 h1, got %d", result.Headings["h1"])
	}
	if result.Headings["h2"] != 2 {
		t.Errorf("expected 2 h2, got %d", result.Headings["h2"])
	}
	if result.Headings["h3"] != 1 {
		t.Errorf("expected 1 h3, got %d", result.Headings["h3"])
	}
}

func TestParseHTMLLinks(t *testing.T) {
	body := `<!DOCTYPE html><html><body>
		<a href="/about">Internal</a>
		<a href="/contact">Internal 2</a>
		<a href="https://external.com/page">External</a>
		<a href="#">Skip</a>
		<a href="javascript:void(0)">Skip JS</a>
	</body></html>`

	base, _ := url.Parse("https://example.com")
	result := &Result{Headings: make(map[string]int)}
	parseHTML(body, base, result)

	if result.InternalLinks != 2 {
		t.Errorf("expected 2 internal links, got %d", result.InternalLinks)
	}
	if result.ExternalLinks != 1 {
		t.Errorf("expected 1 external link, got %d", result.ExternalLinks)
	}
}

func TestLoginFormDetection(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			"login form with password + submit",
			`<form><input type="text" name="email"><input type="password" name="pass"><input type="submit" value="Login"></form>`,
			true,
		},
		{
			"login form with button submit",
			`<form><input type="password"><button type="submit">Sign in</button></form>`,
			true,
		},
		{
			"contact form, no password",
			`<form><input type="text"><textarea></textarea><input type="submit"></form>`,
			false,
		},
		{
			"password field without submit",
			`<form><input type="password"></form>`,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base, _ := url.Parse("https://example.com")
			result := &Result{Headings: make(map[string]int)}
			parseHTML(tc.body, base, result)
			if result.HasLoginForm != tc.expected {
				t.Errorf("expected HasLoginForm=%v, got %v", tc.expected, result.HasLoginForm)
			}
		})
	}
}

func TestAnalyzeInvalidURL(t *testing.T) {
	_, err := Analyze("not a url at all ://")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestAnalyzeHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := Analyze(ts.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	ae, ok := err.(*AnalyzeError)
	if !ok {
		t.Fatalf("expected *AnalyzeError, got %T", err)
	}
	if ae.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", ae.StatusCode)
	}
}

func TestAnalyzeSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Hello World</title></head>
<body>
  <h1>Welcome</h1>
  <h2>Sub 1</h2>
  <h2>Sub 2</h2>
  <a href="/page">Internal</a>
  <a href="https://external.org">External</a>
  <form>
    <input type="email" name="email">
    <input type="password" name="pass">
    <button type="submit">Login</button>
  </form>
</body>
</html>`))
	}))
	defer ts.Close()

	result, err := Analyze(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Hello World" {
		t.Errorf("expected title 'Hello World', got %q", result.Title)
	}
	if result.HTMLVersion != "HTML5" {
		t.Errorf("expected HTML5, got %q", result.HTMLVersion)
	}
	if result.Headings["h1"] != 1 {
		t.Errorf("expected 1 h1")
	}
	if result.Headings["h2"] != 2 {
		t.Errorf("expected 2 h2")
	}
	if result.InternalLinks != 1 {
		t.Errorf("expected 1 internal link, got %d", result.InternalLinks)
	}
	if result.ExternalLinks != 1 {
		t.Errorf("expected 1 external link, got %d", result.ExternalLinks)
	}
	if !result.HasLoginForm {
		t.Error("expected login form to be detected")
	}
}

func TestIsSameHost(t *testing.T) {
	base, _ := url.Parse("https://example.com/page")
	tests := []struct {
		resolved string
		want     bool
	}{
		{"https://example.com/about", true},
		{"https://EXAMPLE.COM/contact", true},
		{"https://sub.example.com/page", false},
		{"https://other.com/page", false},
	}
	for _, tc := range tests {
		got := isSameHost(base, tc.resolved)
		if got != tc.want {
			t.Errorf("isSameHost(%q) = %v, want %v", tc.resolved, got, tc.want)
		}
	}
}

func TestResolveURL(t *testing.T) {
	base, _ := url.Parse("https://example.com/blog/post")
	tests := []struct {
		href string
		want string
	}{
		{"/about", "https://example.com/about"},
		{"../contact", "https://example.com/contact"},
		{"https://other.com", "https://other.com"},
	}
	for _, tc := range tests {
		got := resolveURL(base, tc.href)
		if !strings.HasPrefix(got, tc.want) {
			t.Errorf("resolveURL(%q) = %q, want prefix %q", tc.href, got, tc.want)
		}
	}
}
