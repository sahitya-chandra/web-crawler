# Web Crawler

A simple and extensible web crawler written in Go.

## Overview

This project is a basic web crawler designed to fetch, parse, and archive web pages into a MongoDB database. It demonstrates core crawling techniques such as queue-based URL management, polite crawling, parallel fetching, HTML parsing, and duplicate avoidance.

## Features

- Fetches and parses HTML from web pages
- Extracts page titles, main body content, and hyperlinks
- Enqueues new links for further crawling (breadth-first)
- Polite crawling with delays between requests
- Prevents duplicate crawling using hashed URL tracking
- Stores page data (URL, title, content) in MongoDB
- Modular structure (crawler, queue, db, main)

## Getting Started

### Prerequisites

- Go 1.24+
- A running MongoDB instance (local or remote)
- `git` for cloning the repository

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/sahitya-chandra/web-crawler.git
   cd web-crawler
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up environment variables:**
   - Create a `.env` file in the root directory:
     ```
     MONGODB_URI=mongodb://localhost:27017
     ```

### Usage

1. **Run the crawler:**
   ```bash
   go run main.go
   ```

   By default, the crawler starts at `https://example.com/` and archives up to 500 pages (can be changed in `main.go`).

2. **Database Output:**
   - Crawled web pages are stored in the `crawlerArchive.webpages` collection in MongoDB.
   - Each document contains:
     - `url`: The crawled URL
     - `title`: Page title
     - `content`: Main body content (first 500 words)

### Project Structure

- `main.go` — Entry point; orchestrates queueing, crawling, and database storage
- `crawler/` — Fetches and parses HTML, extracts links and content
- `queue/` — Thread-safe queue implementation for URLs
- `db/` — MongoDB connection and basic storage helpers

### Example Output

```
Crawled: https://example.com/, Title: Example Domain
```

## Configuration

- Adjust the starting URL, maximum pages, or crawling logic in `main.go` as needed.
- MongoDB URI and other secrets are managed via the `.env` file.

## Dependencies

- [Go MongoDB Driver](https://go.mongodb.org/mongo-driver/)
- [godotenv](https://github.com/joho/godotenv)
- [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html)

## License

This project is for educational/demo purposes. No license specified.

## Contributing

Pull requests and suggestions are welcome!

---

> Results may be incomplete. See the [GitHub code search results for more](https://github.com/sahitya-chandra/web-crawler/search?q=crawl+OR+crawler+OR+web+OR+spider+OR+scrape+OR+scraping+OR+url+OR+fetch+OR+parse+OR+download+OR+visit).
