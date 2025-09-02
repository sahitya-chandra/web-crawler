package queue

import (
	"sync"
)

type Queue struct {
	totalQueued int
	length int
	urls []string
	mu sync.Mutex
}

func (q *Queue) Enqueue(url string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.urls = append(q.urls, url)
	q.totalQueued++
	q.length++
}

func (q *Queue) Dequeue() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.urls) == 0 {
		return "", false
	}
	url := q.urls[0]
	q.urls = q.urls[1:]
	q.length--
	return url, true
}

func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.length
}