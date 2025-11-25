package spaserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestServe(t *testing.T) {
	tt := []struct {
		name       string
		url        string
		statusCode int
		body       string
		headers    string
	}{
		{
			name:       "root path renders index.html",
			url:        "http://www.example.com",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		},
		{
			name:       "slash path renders index.html",
			url:        "http://www.example.com/",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		}, {
			name:       "redirects index.html to root",
			url:        "http://www.example.com/index.html",
			statusCode: 301,
			body:       "",
			headers:    "Location:./",
		}, {
			name:       "redirects index.html to root with query string",
			url:        "http://www.example.com/index.html?key1=value",
			statusCode: 301,
			body:       "",
			headers:    "Location:./?key1=value",
		},
		{
			name:       "serve non index file",
			url:        "http://www.example.com/css/main.css",
			statusCode: 200,
			body:       "body {\n\tdisplay: none;\n}",
			headers:    "Accept-Ranges:bytes Content-Length:25 Content-Type:text/css; charset=utf-8",
		},
		{
			name:       "serve root non-index file",
			url:        "http://www.example.com/root-main.css",
			statusCode: 200,
			body:       "body {\n\tdisplay: none;\n}",
			headers:    "Accept-Ranges:bytes Content-Length:25 Content-Type:text/css; charset=utf-8",
		},
		{
			name:       "serves index on file not found",
			url:        "http://www.example.com/doesnotexist.txt",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		},
		{
			name:       "serves index on directory listing",
			url:        "http://www.example.com/css/",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		},
		{
			name:       "path traversal cleaned to safe path",
			url:        "http://www.example.com/../../../etc/passwd",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		},
		{
			name:       "relative traversal cleaned to safe path",
			url:        "http://www.example.com/css/../../../etc/passwd",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		},
		{
			name:       "handles double slashes in path",
			url:        "http://www.example.com//css//main.css",
			statusCode: 200,
			body:       "body {\n\tdisplay: none;\n}",
			headers:    "Accept-Ranges:bytes Content-Length:25 Content-Type:text/css; charset=utf-8",
		},
		{
			name:       "index.html includes security headers",
			url:        "http://www.example.com/",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Security-Policy:default-src 'self' Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0 X-Content-Type-Options:nosniff X-Frame-Options:DENY",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			fsys := os.DirFS("testdata")
			h := Serve(fsys)

			r, err := http.NewRequest(http.MethodGet, tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			body := strings.TrimSpace(w.Body.String())
			statusCode := w.Result().StatusCode

			var headers []string
			for k, v := range w.Result().Header {
				if k == "Last-Modified" {
					continue
				}

				valueString := strings.Join(v, ",")
				headers = append(headers, k+":"+valueString)
			}
			sort.StringSlice(headers).Sort()
			headerString := strings.Join(headers, " ")

			if statusCode != tc.statusCode {
				t.Errorf("statusCode expected: %d, got: %d", tc.statusCode, w.Code)
			}

			if body != tc.body {
				t.Errorf("body expected: %s, got: %s", tc.body, body)
			}

			if headerString != tc.headers {
				t.Errorf("headers expected: %s, got: %s", tc.headers, headerString)
			}
		})
	}
}

func BenchmarkServeStatic(b *testing.B) {
	fsys := os.DirFS("testdata")
	h := Serve(fsys)
	r, _ := http.NewRequest(http.MethodGet, "http://www.example.com/css/main.css", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

func BenchmarkServeIndex(b *testing.B) {
	fsys := os.DirFS("testdata")
	h := Serve(fsys)
	r, _ := http.NewRequest(http.MethodGet, "http://www.example.com/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

func BenchmarkServeNotFound(b *testing.B) {
	fsys := os.DirFS("testdata")
	h := Serve(fsys)
	r, _ := http.NewRequest(http.MethodGet, "http://www.example.com/notfound", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

