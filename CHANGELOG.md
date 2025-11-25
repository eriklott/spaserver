# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2025-11-24

### Added
- Initial release of spaserver package for serving single-page applications
- `Serve()` function that returns an `http.Handler` for serving SPAs from any `fs.FS` filesystem
- Support for both `embed.FS` and `os.DirFS` filesystem implementations
- Automatic fallback to index.html for client-side routing (404s serve index.html)
- Security headers on index.html responses:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Content-Security-Policy: default-src 'self'`
- No-cache headers on index.html to ensure fresh SPA entry points:
  - `Cache-Control: no-cache, no-store, no-transform, must-revalidate, private, max-age=0`
  - `Pragma: no-cache`
  - `Expires` set to Unix epoch
  - `X-Accel-Expires: 0`
- Path traversal protection using `filepath.IsLocal()` validation
- Path normalization using `path.Clean()` to prevent directory traversal attacks
- Automatic redirect from `/index.html` to `/` (preserves query strings)
- Normal browser caching for static assets (CSS, JS, images, etc.)
- Comprehensive test suite with 12 test cases covering:
  - Root path rendering
  - File serving
  - Path traversal protection
  - Directory listing prevention
  - Security header validation
- Benchmark tests for performance measurement of index, static files, and 404 scenarios
- Support for custom `fs.FS` implementations with automatic buffering fallback

### Security
- Path traversal attack prevention through path validation
- Security headers to prevent XSS and clickjacking attacks
- ETag header removal for index.html to prevent cache-related issues
- Protection against directory escape attempts via symlinks (documented caveat for os.DirFS)

[v0.1.0]: https://github.com/eriklott/spaserver/releases/tag/v0.1.0
