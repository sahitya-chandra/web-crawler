package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sahitya-chandra/web-crawler/db"
	"github.com/sahitya-chandra/web-crawler/queue"
	"github.com/sahitya-chandra/web-crawler/crawler"
)

type CrawledSet struct {
	set map[uint64]bool
	length int
	mu sync.RWMutex
}

func (cs *CrawledSet) add(url string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.set[hashUrl(url)] {
		cs.set[hashUrl(url)] = true
		cs.length++
	}
}

func (cs *CrawledSet) contains(url string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.set[hashUrl(url)]
}

func (cs *CrawledSet) size() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.length
}

func hashUrl(url string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(url))

	return h.Sum64()
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI not set in .env file")
	}

	dbInstance, err := db.Connect(uri)
	if err != nil {
		log.Fatal(err)
	}
	defer dbInstance.Disconnect()

	queue := &queue.Queue{}
	crawled := &CrawledSet{set: make(map[uint64]bool)}
	queue.Enqueue("https://www.mjpru.ac.in/")

	fetchChan := make(chan crawler.PageResult, 5)
	parseChan := make(chan crawler.ParsePage, 5)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		crawler.ParseHTML(fetchChan, parseChan, crawled.contains, queue)
	}()

	for queue.Size() > 0 && crawled.size() < 500 {
		url, ok := queue.Dequeue()
		if !ok {
			break
		}

		if crawled.contains(url) {
			continue
		}

		crawled.add(url)

		go crawler.FetchHTML(url, fetchChan)

		parsed := <-parseChan
		if parsed.Err != nil {
			log.Printf("Error parsing %s: %v", parsed.Url, parsed.Err)
            continue
		}

		err = dbInstance.InsertWebpage("webpages", map[string]interface{}{
			"url": parsed.Url,
			"title": parsed.Title,
			"content": strings.ToValidUTF8(parsed.Body, ""),
		})

		if err != nil {
			log.Printf("Error inserting %s: %v", parsed.Url, err)
		}

		fmt.Printf("Crawled: %s, Title: %s\n", parsed.Url, parsed.Title)
        time.Sleep(1 * time.Second) // Polite delay

	}

	
    close(fetchChan)
    close(parseChan)
    wg.Wait()
}