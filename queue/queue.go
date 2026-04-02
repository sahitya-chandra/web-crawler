package queue

import (
	"sync"
)

type Queue struct {
	urls []string
	mu   sync.RWMutex
}

func (q *Queue) Enqueue(url string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.urls = append(q.urls, url)
}

func (q *Queue) Dequeue() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.urls) == 0 {
		return "", false
	}
	url := q.urls[0]
	q.urls = q.urls[1:]
	return url, true
}

func (q *Queue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.urls)
}

func (q *Queue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.urls) == 0
}
