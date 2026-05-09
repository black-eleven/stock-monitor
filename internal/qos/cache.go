package qos

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type klineCacheEntry struct {
	data      []json.RawMessage
	expiresAt time.Time
}

type quoteCacheEntry struct {
	data      *Quote
	expiresAt time.Time
}

type mergeEntry struct {
	result mergeResult
	done   chan struct{}
	once   sync.Once
}

type mergeResult struct {
	data []json.RawMessage
	q    *Quote
	err  error
}

func (m *mergeEntry) resolve(r mergeResult) {
	m.once.Do(func() {
		m.result = r
		close(m.done)
	})
}

func cacheTTL(kt int) time.Duration {
	if kt <= 240 {
		return 30 * time.Second
	}
	return 5 * time.Minute
}

var (
	klineCache   = map[string]*klineCacheEntry{}
	klineCacheMu sync.RWMutex
	klineMerge   = map[string]*mergeEntry{}
	klineMergeMu sync.Mutex
)

var (
	quoteCache   = map[string]*quoteCacheEntry{}
	quoteCacheMu sync.RWMutex
	quoteMerge   = map[string]*mergeEntry{}
	quoteMergeMu sync.Mutex
)

func klineCacheKey(code string, kt, count int) string {
	return fmt.Sprintf("%s:%d:%d", code, kt, count)
}

func isRetryable(err error) bool {
	s := err.Error()
	return s == "not connected" || s == "connection lost" ||
		strings.Contains(s, "request timeout")
}

// fetchKlineWithRetry retries FetchHistoryKline up to 3 times on transient errors.
func (c *QosClient) fetchKlineWithRetry(code string, kt, count int) ([]json.RawMessage, error) {
	var lastErr error
	delay := 1 * time.Second
	for i := 0; i < 3; i++ {
		data, err := c.FetchHistoryKline(code, kt, count)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !isRetryable(err) {
			break
		}
		time.Sleep(delay)
		delay *= 2
	}
	return nil, lastErr
}

func (c *QosClient) FetchHistoryKlineCached(code string, kt int, count int) ([]json.RawMessage, error) {
	key := klineCacheKey(code, kt, count)

	klineCacheMu.RLock()
	if e, ok := klineCache[key]; ok && time.Now().Before(e.expiresAt) {
		data := e.data
		klineCacheMu.RUnlock()
		return data, nil
	}
	klineCacheMu.RUnlock()

	klineMergeMu.Lock()
	m, exists := klineMerge[key]
	if !exists {
		m = &mergeEntry{done: make(chan struct{})}
		klineMerge[key] = m
	}
	klineMergeMu.Unlock()

	if !exists {
		data, err := c.fetchKlineWithRetry(code, kt, count)
		if err == nil {
			klineCacheMu.Lock()
			klineCache[key] = &klineCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL(kt))}
			klineCacheMu.Unlock()
		}
		m.resolve(mergeResult{data: data, err: err})
		klineMergeMu.Lock()
		delete(klineMerge, key)
		klineMergeMu.Unlock()
		return data, err
	}

	<-m.done
	return m.result.data, m.result.err
}

// fetchQuoteWithRetry retries FetchQuote up to 3 times on transient errors.
func (c *QosClient) fetchQuoteWithRetry(code string) (*Quote, error) {
	var lastErr error
	delay := 1 * time.Second
	for i := 0; i < 3; i++ {
		q, err := c.FetchQuote(code)
		if err == nil {
			return q, nil
		}
		lastErr = err
		if !isRetryable(err) {
			break
		}
		time.Sleep(delay)
		delay *= 2
	}
	return nil, lastErr
}

func (c *QosClient) FetchQuoteCached(code string) (*Quote, error) {
	key := code

	quoteCacheMu.RLock()
	if e, ok := quoteCache[key]; ok && time.Now().Before(e.expiresAt) {
		q := e.data
		quoteCacheMu.RUnlock()
		return q, nil
	}
	quoteCacheMu.RUnlock()

	quoteMergeMu.Lock()
	m, exists := quoteMerge[key]
	if !exists {
		m = &mergeEntry{done: make(chan struct{})}
		quoteMerge[key] = m
	}
	quoteMergeMu.Unlock()

	if !exists {
		q, err := c.fetchQuoteWithRetry(code)
		if err == nil && q != nil {
			quoteCacheMu.Lock()
			quoteCache[key] = &quoteCacheEntry{data: q, expiresAt: time.Now().Add(30 * time.Second)}
			quoteCacheMu.Unlock()
		}
		m.resolve(mergeResult{q: q, err: err})
		quoteMergeMu.Lock()
		delete(quoteMerge, key)
		quoteMergeMu.Unlock()
		return q, err
	}

	<-m.done
	return m.result.q, m.result.err
}
