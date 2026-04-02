package queue

import (
	"sync"
	"testing"
)

func TestEnqueueDequeue(t *testing.T) {
	q := &Queue{}
	q.Enqueue("https://a.com")
	q.Enqueue("https://b.com")

	if q.Size() != 2 {
		t.Fatalf("expected size 2, got %d", q.Size())
	}

	url, ok := q.Dequeue()
	if !ok || url != "https://a.com" {
		t.Fatalf("expected https://a.com, got %s (ok=%v)", url, ok)
	}

	url, ok = q.Dequeue()
	if !ok || url != "https://b.com" {
		t.Fatalf("expected https://b.com, got %s (ok=%v)", url, ok)
	}

	_, ok = q.Dequeue()
	if ok {
		t.Fatal("expected Dequeue to return false on empty queue")
	}
}

func TestIsEmpty(t *testing.T) {
	q := &Queue{}
	if !q.IsEmpty() {
		t.Fatal("new queue should be empty")
	}

	q.Enqueue("https://a.com")
	if q.IsEmpty() {
		t.Fatal("queue with item should not be empty")
	}

	q.Dequeue()
	if !q.IsEmpty() {
		t.Fatal("queue should be empty after dequeue")
	}
}

func TestFIFOOrder(t *testing.T) {
	q := &Queue{}
	urls := []string{"https://1.com", "https://2.com", "https://3.com", "https://4.com"}

	for _, u := range urls {
		q.Enqueue(u)
	}

	for _, expected := range urls {
		got, ok := q.Dequeue()
		if !ok {
			t.Fatal("unexpected empty queue")
		}
		if got != expected {
			t.Fatalf("expected %s, got %s", expected, got)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	q := &Queue{}
	var wg sync.WaitGroup
	n := 100

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			q.Enqueue("https://example.com")
		}(i)
	}
	wg.Wait()

	if q.Size() != n {
		t.Fatalf("expected size %d, got %d", n, q.Size())
	}

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			q.Dequeue()
		}()
	}
	wg.Wait()

	if !q.IsEmpty() {
		t.Fatalf("expected empty queue, got size %d", q.Size())
	}
}
