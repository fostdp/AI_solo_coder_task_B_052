package influx

import (
	"log"
	"sync"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

type writeRequest struct {
	bp      influxdb.BatchPoints
	attempt int
}

type AsyncWriter struct {
	client     influxdb.Client
	queue      chan writeRequest
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	wg         sync.WaitGroup
	stopCh     chan struct{}
	dropped    uint64
	retried    uint64
	mu         sync.Mutex
}

func NewAsyncWriter(client influxdb.Client, queueSize int, maxRetries int) *AsyncWriter {
	if queueSize <= 0 {
		queueSize = 4096
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	aw := &AsyncWriter{
		client:     client,
		queue:      make(chan writeRequest, queueSize),
		maxRetries: maxRetries,
		baseDelay:  500 * time.Millisecond,
		maxDelay:   30 * time.Second,
		stopCh:     make(chan struct{}),
	}
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		aw.wg.Add(1)
		go aw.worker()
	}
	return aw
}

func (aw *AsyncWriter) Write(bp influxdb.BatchPoints) bool {
	select {
	case aw.queue <- writeRequest{bp: bp, attempt: 0}:
		return true
	default:
		aw.mu.Lock()
		aw.dropped++
		aw.mu.Unlock()
		log.Printf("[AsyncWriter] queue full, dropping batch (total dropped: %d)", aw.dropped)
		return false
	}
}

func (aw *AsyncWriter) WriteSync(bp influxdb.BatchPoints) error {
	return aw.writeWithRetry(bp, 0)
}

func (aw *AsyncWriter) worker() {
	defer aw.wg.Done()
	for {
		select {
		case <-aw.stopCh:
			return
		case req := <-aw.queue:
			err := aw.writeWithRetry(req.bp, req.attempt)
			if err != nil {
				if req.attempt < aw.maxRetries {
					aw.requeue(req)
				} else {
					log.Printf("[AsyncWriter] permanently failed after %d retries: %v", req.attempt, err)
				}
			}
		}
	}
}

func (aw *AsyncWriter) writeWithRetry(bp influxdb.BatchPoints, attempt int) error {
	err := aw.client.Write(bp)
	if err == nil {
		return nil
	}

	if attempt >= aw.maxRetries {
		return err
	}

	delay := aw.baseDelay * time.Duration(1<<uint(attempt))
	if delay > aw.maxDelay {
		delay = aw.maxDelay
	}

	log.Printf("[AsyncWriter] write failed (attempt %d/%d), retrying in %v: %v",
		attempt+1, aw.maxRetries, delay, err)

	time.Sleep(delay)

	return aw.writeWithRetry(bp, attempt+1)
}

func (aw *AsyncWriter) requeue(req writeRequest) {
	aw.mu.Lock()
	aw.retried++
	aw.mu.Unlock()

	req.attempt++
	select {
	case aw.queue <- req:
	default:
		log.Printf("[AsyncWriter] requeue failed, queue full (attempt %d)", req.attempt)
	}
}

func (aw *AsyncWriter) Close() {
	close(aw.stopCh)
	done := make(chan struct{})
	go func() {
		aw.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		log.Println("[AsyncWriter] workers did not stop in time, forcing exit")
	}
}

func (aw *AsyncWriter) Stats() map[string]interface{} {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	return map[string]interface{}{
		"queue_len":    len(aw.queue),
		"queue_cap":    cap(aw.queue),
		"dropped":      aw.dropped,
		"retried":      aw.retried,
		"max_retries":  aw.maxRetries,
	}
}
