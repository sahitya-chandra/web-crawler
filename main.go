package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sahitya-chandra/web-crawler/crawler"
	"github.com/sahitya-chandra/web-crawler/db"
	"github.com/sahitya-chandra/web-crawler/queue"
)

type config struct {
	startURL   string
	maxPages   int
	delay      time.Duration
	database   string
	collection string
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.startURL, "url", "https://example.com/", "starting URL to crawl")
	flag.IntVar(&cfg.maxPages, "max", 500, "maximum number of pages to crawl")
	flag.DurationVar(&cfg.delay, "delay", 1*time.Second, "delay between requests (e.g. 500ms, 2s)")
	flag.StringVar(&cfg.database, "db", "crawlerArchive", "MongoDB database name")
	flag.StringVar(&cfg.collection, "collection", "webpages", "MongoDB collection name")
	flag.Parse()
	return cfg
}

// visited tracks which URLs have already been crawled using a concurrent-safe set.
type visited struct {
	set map[string]bool
	mu  sync.RWMutex
}

func (v *visited) add(url string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.set[url] = true
}

func (v *visited) contains(url string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.set[url]
}

func (v *visited) size() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.set)
}

func main() {
	cfg := parseFlags()

	_ = godotenv.Load()

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI not set — create a .env file or export the variable")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	dbInstance, err := db.Connect(ctx, uri, cfg.database)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer func() {
		if err := dbInstance.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting: %v", err)
		}
	}()

	crawl(ctx, cfg, dbInstance)
}

func crawl(ctx context.Context, cfg config, dbInstance *db.DB) {
	q := &queue.Queue{}
	seen := &visited{set: make(map[string]bool)}

	q.Enqueue(cfg.startURL)

	crawled := 0

	for !q.IsEmpty() && crawled < cfg.maxPages {
		if ctx.Err() != nil {
			log.Println("Crawl cancelled")
			break
		}

		url, ok := q.Dequeue()
		if !ok {
			break
		}

		if seen.contains(url) {
			continue
		}

		result := crawler.FetchHTML(ctx, url)

		seen.add(url)

		parsed := crawler.ParseHTML(result)
		if parsed.Err != nil {
			log.Printf("Skipped %s: %v", parsed.URL, parsed.Err)
			continue
		}

		page := db.Webpage{
			URL:     parsed.URL,
			Title:   parsed.Title,
			Content: strings.ToValidUTF8(parsed.Body, ""),
		}
		if err := dbInstance.InsertWebpage(ctx, cfg.collection, page); err != nil {
			log.Printf("Insert failed for %s: %v", parsed.URL, err)
		}

		for _, link := range parsed.Links {
			if !seen.contains(link) {
				q.Enqueue(link)
			}
		}

		crawled++
		fmt.Printf("[%d/%d] %s — %s\n", crawled, cfg.maxPages, parsed.URL, parsed.Title)

		select {
		case <-ctx.Done():
			log.Println("Crawl cancelled")
			return
		case <-time.After(cfg.delay):
		}
	}

	log.Printf("Done. Crawled %d pages.", crawled)
}
