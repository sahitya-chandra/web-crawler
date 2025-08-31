package main

import (
	"fmt"
	"hash/fnv"
	"sync"
	"net/http"


)

type CrawledSet struct {
	set map[uint64]bool
	length int
	mu sync.RWMutex
}

func (cs *CrawledSet) add(url string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.set[hashUrl(url)] = true
	cs.length++
}

func (cs *CrawledSet) contains(url string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.set[hashUrl(url)]
}

func (cs *CrawledSet) size() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return cs.length
}

func hashUrl(url string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(url))

	return h.Sum64()
}



func main() {
	fmt.Println("hd")
}