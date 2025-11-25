package spaserver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
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

var securityHeaders = map[string]string{
	"X-Content-Type-Options":  "nosniff",
	"X-Frame-Options":         "DENY",
	"Content-Security-Policy": "default-src 'self'",
}

// Serve a single-page application from the filesystem.
//
// SECURITY NOTES:
//   - When using os.DirFS: Symlinks are followed and may escape the root directory.
//     For untrusted filesystems, consider using Go 1.24+ os.Root instead.
//   - When using embed.FS: Symlinks are not supported (build-time only).
//   - All paths are cleaned using path.Clean to prevent basic traversal attacks.
//   - Path validation using filepath.IsLocal prevents directory traversal attempts.
//
// BEHAVIOR:
// - Requests for /index.html redirect to /
// - Requests for / or non-existent files serve index.html
// - index.html responses include no-cache and security headers
// - Other files are cached normally
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

		// Validate the path is safe (prevents directory traversal)
		if !filepath.IsLocal(name) {
			serveError(w, "400 Bad Request", http.StatusBadRequest)
			return
		}

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

// serveIndex sends the index.html file with no-cache and security headers.
// This prevents caching of the SPA entry point, ensuring users always get
// the latest version and route handling works correctly.
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

	// Set security headers
	for k, v := range securityHeaders {
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

// fileToReadSeeker converts an fs.File to io.ReadSeeker for http.ServeContent.
// embed.FS and os.DirFS files implement io.ReadSeeker directly.
// Custom fs.FS implementations may need buffering as a fallback.
func fileToReadSeeker(file fs.File) (io.ReadSeeker, error) {
	// Try to assert to io.ReadSeeker
	// Both embed.FS and os.DirFS files implement this
	if seeker, ok := file.(io.ReadSeeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek failed: %w", err)
		}
		return seeker, nil
	}

	// Fallback: buffer it (for non-standard fs.FS implementations)
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return bytes.NewReader(data), nil
}
