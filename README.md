# Web Crawler

A simple web crawler written in Go that fetches, parses, and archives web pages into MongoDB.

Built as a learning project to explore Go's concurrency primitives, HTTP handling, and HTML parsing.

## Setup

Requires **Go 1.24+** and a running **MongoDB** instance.

```bash
git clone https://github.com/sahitya-chandra/web-crawler.git
cd web-crawler
go mod download
```

Create a `.env` file:

```
MONGODB_URI=mongodb://localhost:27017
```

## Usage

```bash
go run main.go                                    # defaults: example.com, 500 pages
go run main.go -url https://go.dev/ -max 100      # custom URL and limit
go run main.go -delay 500ms                       # faster crawling
```

| Flag          | Default                | Description                      |
|---------------|------------------------|----------------------------------|
| `-url`        | `https://example.com/` | Starting URL to crawl            |
| `-max`        | `500`                  | Maximum pages to crawl           |
| `-delay`      | `1s`                   | Delay between requests           |
| `-db`         | `crawlerArchive`       | MongoDB database name            |
| `-collection` | `webpages`             | MongoDB collection name          |

## How It Works

1. Dequeues a URL from a FIFO queue
2. Fetches the page via HTTP (with timeout and `User-Agent`)
3. Parses HTML to extract the title, body text (first 500 words), and links
4. Stores the page in MongoDB
5. Enqueues discovered links that haven't been seen yet
6. Waits for the configured delay, then repeats

Stops when the queue is empty, the page limit is reached, or `Ctrl+C` is pressed.

## Project Structure

```
main.go              Entry point — CLI flags, crawl loop, graceful shutdown
crawler/crawler.go   HTTP fetching, HTML parsing, link normalization
queue/queue.go       Thread-safe FIFO URL queue
db/db.go             MongoDB connection and document insertion
```

## Make Targets

```bash
make build    # compile binary
make run      # build and run
make test     # run tests with race detector  (go test -race ./...)
make fmt      # format code
make check    # fmt + vet + test
make clean    # remove binary
```
