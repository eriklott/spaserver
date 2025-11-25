# spaserver

A production-ready Go HTTP handler for serving Single Page Applications (SPAs) from any `fs.FS` filesystem.

## Features

- **SPA Routing**: Automatically serves `index.html` for non-existent routes (enables client-side routing)
- **Security Headers**: Adds `X-Content-Type-Options`, `X-Frame-Options`, and `Content-Security-Policy` headers
- **Smart Caching**: No-cache headers for `index.html`, normal caching for static assets
- **Path Traversal Protection**: Built-in validation to prevent directory traversal attacks
- **Flexible**: Works with `os.DirFS`, `embed.FS`, or any custom `fs.FS` implementation
- **Zero Dependencies**: Uses only the Go standard library

## Installation

```bash
go get github.com/eriklott/spaserver
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/eriklott/spaserver"
)

func main() {
    // Serve a SPA from the ./dist directory
    // Expects index.html to be at ./dist/index.html
    fsys := os.DirFS("dist")
    handler := spaserver.Serve(fsys)

    log.Println("Server running on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

**Important**: The filesystem provided must have `index.html` at its root. For example, if using `os.DirFS("dist")`, your directory structure should be:
```
dist/
├── index.html
├── css/
├── js/
└── ...
```

## Usage Examples

### Example 1: Serving from a Directory

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/eriklott/spaserver"
)

func main() {
    // Serve files from the ./public directory
    fsys := os.DirFS("public")

    http.Handle("/", spaserver.Serve(fsys))

    log.Println("Serving SPA on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Example 2: Using Embedded Files

```go
package main

import (
    "embed"
    "log"
    "net/http"

    "github.com/eriklott/spaserver"
)

//go:embed dist/*
var staticFiles embed.FS

func main() {
    // Serve embedded files from the dist subdirectory
    fsys, err := fs.Sub(staticFiles, "dist")
    if err != nil {
        log.Fatal(err)
    }

    http.Handle("/", spaserver.Serve(fsys))

    log.Println("Serving embedded SPA on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Example 3: Multiple SPAs with Different Routes

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/eriklott/spaserver"
)

func main() {
    // Serve admin SPA at /admin
    adminFS := os.DirFS("admin-dist")
    http.Handle("/admin/", http.StripPrefix("/admin", spaserver.Serve(adminFS)))

    // Serve main SPA at root
    mainFS := os.DirFS("main-dist")
    http.Handle("/", spaserver.Serve(mainFS))

    log.Println("Serving multiple SPAs on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## How It Works

The handler implements SPA-friendly routing behavior:

1. **Root path (`/`)**: Serves `index.html` with no-cache headers
2. **Static assets (`/css/style.css`)**: Serves the file normally with standard caching
3. **Non-existent routes (`/about`, `/user/123`)**: Serves `index.html` (enables client-side routing)
4. **index.html requests**: Redirects `/index.html` to `/` to prevent duplicate content
5. **Security**: All paths are validated to prevent directory traversal attacks

### Headers Behavior

**For `index.html`:**
- `Cache-Control: no-cache, no-store, must-revalidate`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Content-Security-Policy: default-src 'self'`

**For static assets:**
- Standard HTTP caching (uses `Last-Modified` and `ETag`)
- No security headers (allows customization)

## Security Considerations

This package includes several security features:

1. **Path Traversal Protection**: Uses `path.Clean()` and `filepath.IsLocal()` to prevent `../` attacks
2. **Security Headers**: Adds protection against XSS, clickjacking, and MIME sniffing
3. **Content Security Policy**: Default CSP restricts resources to same origin

### Important Security Notes

- When using `os.DirFS`, symlinks are followed and may escape the root directory. For untrusted filesystems, consider using Go 1.24+ `os.Root` instead.
- When using `embed.FS`, symlinks are not supported (build-time only).
- The default CSP (`default-src 'self'`) may be too restrictive for some applications. You can override headers after they're set if needed.

## API

### `func Serve(fsys fs.FS) http.Handler`

Creates an HTTP handler that serves a Single Page Application from the provided filesystem.

**Parameters:**
- `fsys fs.FS`: Any filesystem implementing `fs.FS` (e.g., `os.DirFS`, `embed.FS`)

**Returns:**
- `http.Handler`: An HTTP handler that can be used with `http.Handle()` or `http.ListenAndServe()`

**Behavior:**
- Requests for `/` or non-existent files serve `index.html`
- Requests for `/index.html` redirect to `/`
- `index.html` is served with no-cache and security headers
- Other files are cached normally
- Directory listings serve `index.html` instead

## License

MIT
