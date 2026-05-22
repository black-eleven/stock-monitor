package eastmoney

import (
	"encoding/json"
	"fmt"
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
	fund *Fundamentals
	err  error
}

func (m *mergeEntry) resolve(r mergeResult) {
	m.once.Do(func() {
		m.result = r
		close(m.done)
	})
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

func cacheTTL(kt int) time.Duration {
	if kt <= 240 {
		return 30 * time.Second
	}
	return 5 * time.Minute
}

func (c *Client) FetchHistoryKlineCached(code string, kt int, count int) ([]json.RawMessage, error) {
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
		data, err := c.FetchHistoryKline(code, kt, count)
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

type fundamentalsCacheEntry struct {
	data      *Fundamentals
	expiresAt time.Time
}

var (
	fundamentalsCache   = map[string]*fundamentalsCacheEntry{}
	fundamentalsCacheMu sync.RWMutex
	fundamentalsMerge   = map[string]*mergeEntry{}
	fundamentalsMergeMu sync.Mutex
)

func (c *Client) FetchFundamentalsCached(code string) (*Fundamentals, error) {
	key := code

	fundamentalsCacheMu.RLock()
	if e, ok := fundamentalsCache[key]; ok && time.Now().Before(e.expiresAt) {
		d := e.data
		fundamentalsCacheMu.RUnlock()
		return d, nil
	}
	var stale *Fundamentals
	if e, ok := fundamentalsCache[key]; ok {
		stale = e.data
	}
	fundamentalsCacheMu.RUnlock()

	fundamentalsMergeMu.Lock()
	m, exists := fundamentalsMerge[key]
	if !exists {
		m = &mergeEntry{done: make(chan struct{})}
		fundamentalsMerge[key] = m
	}
	fundamentalsMergeMu.Unlock()

	if !exists {
		f, err := c.FetchFundamentals(code)
		if err == nil && f != nil {
			fundamentalsCacheMu.Lock()
			fundamentalsCache[key] = &fundamentalsCacheEntry{
				data: f, expiresAt: time.Now().Add(5 * time.Minute),
			}
			fundamentalsCacheMu.Unlock()
		} else if err != nil && stale != nil {
			f = stale
			err = nil
		}
		m.resolve(mergeResult{fund: f, err: err})
		fundamentalsMergeMu.Lock()
		delete(fundamentalsMerge, key)
		fundamentalsMergeMu.Unlock()
		return f, err
	}

	<-m.done
	return m.result.fund, m.result.err
}

func (c *Client) FetchQuoteCached(code string) (*Quote, error) {
	key := code

	quoteCacheMu.RLock()
	if e, ok := quoteCache[key]; ok && time.Now().Before(e.expiresAt) {
		q := e.data
		quoteCacheMu.RUnlock()
		return q, nil
	}
	var stale *Quote
	if e, ok := quoteCache[key]; ok {
		stale = e.data
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
		q, err := c.FetchQuote(code)
		if err == nil && q != nil {
			quoteCacheMu.Lock()
			quoteCache[key] = &quoteCacheEntry{data: q, expiresAt: time.Now().Add(30 * time.Second)}
			quoteCacheMu.Unlock()
		} else if err != nil && stale != nil {
			q = stale
			err = nil
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
