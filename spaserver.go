package spaserver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	"path"
	"strings"
	"time"
)

const indexPage = "index.html"

// Unix epoch time
var epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

// Taken from https://github.com/mytrile/nocache
var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// Serve a single-page application from the filesystem. Requests for `index.html` will be redirected to the root path.
// Requests for the root path or non-existent files will render the `index.html` page. When the index page is rendered,
// headers are set to prevent caching by upstream servers.
func Serve(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Normalize and clean the path
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		upath = path.Clean(upath)

		// redirect .../index.html to .../
		// can't use Redirect() because that would make the path absolute,
		// which would be a problem running under StripPrefix
		if strings.HasSuffix(r.URL.Path, "/"+indexPage) {
			localRedirect(w, r, "./")
			return
		}

		// Serve index page on root path
		if upath == "/" {
			serveIndex(fsys, w, r)
			return
		}

		name := strings.TrimPrefix(upath, "/")
		file, err := fsys.Open(name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				serveIndex(fsys, w, r)
				return
			}
			if errors.Is(err, fs.ErrPermission) {
				serveError(w, "403 Forbidden", http.StatusForbidden)
				return
			}
			// Default:
			serveError(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		fstat, err := file.Stat()
		if err != nil {
			serveError(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		// If the path is a directory, display the index html page instead
		if fstat.IsDir() {
			serveIndex(fsys, w, r)
			return
		}

		seeker, err := fileToReadSeeker(file)
		if err != nil {
			serveError(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Serve the content
		http.ServeContent(w, r, path.Base(upath), fstat.ModTime(), seeker)
	})
}

// serveIndex sets headers to prevent caching by upstream servers.
func serveIndex(fsys fs.FS, w http.ResponseWriter, r *http.Request) {
	b, err := fs.ReadFile(fsys, indexPage)
	if err != nil {
		serveError(w, "404 Page Not Found", http.StatusNotFound)
		return
	}

	seeker := bytes.NewReader(b)

	// Delete any ETag headers that may have been set
	for _, v := range etagHeaders {
		if r.Header.Get(v) != "" {
			r.Header.Del(v)
		}
	}

	// Set NoCache headers
	for k, v := range noCacheHeaders {
		w.Header().Set(k, v)
	}

	http.ServeContent(w, r, indexPage, time.Unix(0, 0), seeker)
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(w http.ResponseWriter, r *http.Request, newPath string) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusMovedPermanently)
}

// serveError serves an error from ServeFile, ServeFileFS, and ServeContent.
// Because those can all be configured by the caller by setting headers like
// Etag, Last-Modified, and Cache-Control to send on a successful response,
// the error path needs to clear them, since they may not be meant for errors.
func serveError(w http.ResponseWriter, text string, code int) {
	h := w.Header()

	for _, k := range []string{
		"Cache-Control",
		"Content-Encoding",
		"Etag",
		"Last-Modified",
	} {
		if _, hasKey := h[k]; !hasKey {
			continue
		}
		h.Del(k)
	}

	http.Error(w, text, code)
}

func fileToReadSeeker(file fs.File) (io.ReadSeeker, error) {
	// Try to assert to io.ReadSeeker
	if seeker, ok := file.(io.ReadSeeker); ok {
		// Can seek directly
		seeker.Seek(0, io.SeekStart)
		return seeker, nil
	}

	// Otherwise, buffer it
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read all: %w", err)
	}

	seeker := bytes.NewReader(data)

	return seeker, nil
}
