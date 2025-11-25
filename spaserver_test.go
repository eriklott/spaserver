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
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0",
		},
		{
			name:       "slash path renders index.html",
			url:        "http://www.example.com/",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0",
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
			body:       "css/main.css",
			headers:    "Accept-Ranges:bytes Content-Length:13 Content-Type:text/css; charset=utf-8",
		},
		{
			name:       "serve root non-index file",
			url:        "http://www.example.com/root-main.css",
			statusCode: 200,
			body:       "root-main.css",
			headers:    "Accept-Ranges:bytes Content-Length:14 Content-Type:text/css; charset=utf-8",
		},
		{
			name:       "serves index on file not found",
			url:        "http://www.example.com/doesnotexist.txt",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0",
		},
		{
			name:       "serves index on directory listing",
			url:        "http://www.example.com/css/",
			statusCode: 200,
			body:       "index.html",
			headers:    "Accept-Ranges:bytes Cache-Control:no-cache, no-store, no-transform, must-revalidate, private, max-age=0 Content-Length:11 Content-Type:text/html; charset=utf-8 Expires:Thu, 01 Jan 1970 00:00:00 GMT Pragma:no-cache X-Accel-Expires:0",
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
