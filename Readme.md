# caching-proxy

> An HTTP reverse proxy with an in-memory LRU cache.

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)


## Table of Contents

- [Introduction](#introduction)
- [How It Works](#how-it-works)
- [Caching Concepts & Implementation](#caching-concepts--implementation)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Project Structure](#project-structure)
- [Usage](#usage)
  - [Start the Proxy](#start-the-proxy)
  - [Test Caching](#test-caching)
  - [Clear the Cache](#clear-the-cache)
- [CLI Reference](#cli-reference)
- [Deployment](#deployment)


## Introduction

Every HTTP API receives repeated requests for the same data. Without caching, the origin server processes each one from scratch — wasting resources and increasing latency for users.

`caching-proxy` sits **between** the client and the origin server. It forwards requests the first time, caches the response, and serves subsequent identical requests directly from memory. The result: faster responses and reduced load on the origin.


## How It Works

```
  ┌──────────┐                     ┌────────────────┐                     ┌──────────────┐
  │  Client  │     GET /users      │ caching-proxy  │    GET /users       │    Origin    │
  │ (curl/   │ ──────────────────→ │                │ ──────────────────→ │              │
  │ browser) │                     │  (port 8000)   │                     │ (httpbin.io) │
  └──────────┘                     └────────────────┘                     └──────────────┘
                                        │                                       │
                                   ┌────┴─────┐                                 │
                                   │  Is it a │                                 │
                                   │ GET req? │                                 │
                                   └────┬─────┘                                 │
                                    YES │  NO                                   │
                                   ┌────┴──────┐                                │
                                   │           │                                │
                                   ▼           ▼                                │
                             ┌──────────┐  ┌──────────┐                         │
                             │  Check   │  │ Forward  │                         │
                             │  Cache   │  │ directly │                         │
                             └────┬─────┘  └─────┬────┘                         │
                              ┌───┴─────┐        │                              │
                              │         │        │                              │ 
                              ▼         ▼        │                              │
                         ┌────────┐ ┌────────┐   │                              │
                         │  HIT   │ │  MISS  │   │                              │
                         └───┬────┘ └───┬────┘   │                              │ 
                             │          │        │                              │
                             ▼          ▼        │                              │
                       ┌──────────┐ ┌────────┐   │                              │
                       │ Serve    │ │Forward │   │                              │
                       │ cached   │ │to      │───┼─────────────────────────────→│
                       │ response │ │origin  │   │                              │
                       │ +X-Cache:│ └───┬────┘   │                              │
                       │ HIT      │     │        │                              │
                       └──────────┘     │        │                              │
                                        ▼        │                              │
                                  ┌──────────┐   │                              │
                                  │ Cache    │   │                              │
                                  │ response │   │                              │
                                  │ (if 2xx, │   │                              │
                                  │ no Auth, │   │                              │
                                  │ no Set-  │   │                              │
                                  │ Cookie)  │   │                              │
                                  └──────────┘   │                              │
                                        │        │                              │
                                        ▼        ▼                              │
                                  ┌────────────────────┐                        │
                                  │ Response to Client │ ◄──────────────────────┘
                                  └────────────────────┘
```


## Caching Concepts & Implementation

| Concept | What It Means | How I Implemented It |
|---|---|---|
| **Reverse Proxy** | A server that sits between clients and a backend, forwarding requests and returning responses. Go's `httputil.ReverseProxy` handles the low-level details like connection pooling and error handling. | I constructed a `ReverseProxy` directly using the `Rewrite` field (the modern, secure replacement for the deprecated `Director`). The `Rewrite` function rewrites the outbound URL to the origin, preserves the `Host` header, and forwards `X-Forwarded-*` headers so the origin sees the real client IP. |
| **LRU Eviction** | When the cache reaches its size limit, the **L**east **R**ecently **U**sed entry is removed to make room for new ones. This keeps frequently accessed data in cache. | I used Go's `container/list` (a doubly linked list) combined with a `map[string]*list.Element`. Each cache access moves the item to the **front** of the list. When evicting, I remove from the **back** — that's the least recently used entry. Both operations are O(1). |
| **TTL (Time To Live)** | Cached items expire after a fixed duration, ensuring stale data is eventually refreshed. | Each `CacheItem` stores an `expiration` timestamp set to `time.Now().Add(5 * time.Minute)`. On every `Get`, I check if the current time has passed the expiration. Expired items are removed from both the list and the map. |
| **Cache Key** | A unique identifier for each cached response. The same key must produce the same response. | Used `r.URL.String()`: The full request URL including query parameters. Two requests to `GET /api/users?page=1` and `GET /api/users?page=2` are cached separately. |
| **X-Cache Header** | A standard header (`X-Cache: HIT` or `X-Cache: MISS`) that tells the client whether the response came from cache or the origin. Useful for debugging and monitoring. | On a hit, I set `X-Cache: HIT` on the response. On a miss, the middleware forwards the request and sets `X-Cache: MISS` on the outgoing request to the origin (which is captured and stored with the cached response). |
| **Security Exclusion** | Authenticated responses should never be cached and served to other users. | I check for the `Authorization` header on the incoming request. If present, the response is forwarded but **not** cached. |
| **Cookie Exclusion** | Responses containing `Set-Cookie` should not be cached, as cookies are user-specific. | I check for the `Set-Cookie` header on the origin's response. If present, the response is **not** stored in the cache. |
| **Admin Endpoint** | A way to interact with the cache at runtime without restarting the proxy. | I register `/admin/clear-cache` on the same HTTP server via a `ServeMux`. The admin routes are handled separately from the proxy logic. The `cache-clear` CLI subcommand sends an HTTP POST to this endpoint. |

## Features

- **In-memory LRU cache** with configurable size limit
- **TTL-based expiration** - items expire after 5 minutes
- **`X-Cache: HIT` / `X-Cache: MISS`** headers for debugging
- **Admin endpoint** (`/admin/clear-cache`) to flush the cache at runtime
- **`cache-clear`** CLI subcommand for convenience
- **Security-aware** - skips caching for authenticated requests (`Authorization` header)
- **Cookie-aware** - skips caching for responses with `Set-Cookie` headers
- **CLI flags** for port, origin, and cache size

## Prerequisites

- [Go](https://go.dev/dl/) 1.26 or later


## Installation

```bash
# Clone the repository
git clone https://github.com/<your-username>/caching-proxy.git
cd caching-proxy

# Build the binary
go build -o caching-proxy
```

## Project Structure

```
caching-proxy/
├── main.go              # Entry point — delegates to cmd.Execute()
├── cmd/
│   ├── root.go          # CLI commands, flags, and validation
│   ├── proxy.go         # Reverse proxy setup (Rewrite, ServeMux, admin route)
│   └── cache.go         # LRU cache, ResponseInterceptor, middleware, cache-clear logic
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── .gitignore
├── LICENSE
└── README.md
```

## Usage

### Start the Proxy

```bash
./caching-proxy --port 8000 --origin https://httpbin.org --maxsize 500
```

This starts the proxy on port `8000`, forwarding all requests to `https://httpbin.org`, with an LRU cache that holds at most 500 entries.

### Test Caching

```bash
# First request — cache miss
curl -i http://localhost:8000/get
# Look for: X-Cache: MISS

# Second request to the same URL — cache hit
curl -i http://localhost:8000/get
# Look for: X-Cache: HIT

# Different URL — cache miss
curl -i http://localhost:8000/uuid
```

### Clear the Cache

```bash
# While the proxy is running, in a separate terminal:
./caching-proxy cache-clear --port 8000
# Output: Cache cleared

# The next request will be a cache miss again
curl -i http://localhost:8000/get
# Look for: X-Cache: MISS
```

## CLI Reference

### `caching-proxy` (start the server)

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--port` | `-p` | `8000` | Port on which to run the proxy server |
| `--origin` | `-o` | `""` | The URL of the server to proxy to **(required)** |
| `--maxsize` | `-s` | `1000` | Maximum number of items in the cache (clamped to 1000) |

### `caching-proxy cache-clear`

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--port` | `-p` | `8000` | Port of the running proxy instance |

## Deployment

This project is deployed on [Render's free tier](https://render.com). Note that free instances spin down after periods of inactivity, so the cache is lost on wake.


This project was made with inspiration from [Roadmap](https://roadmap.sh/projects/caching-server).