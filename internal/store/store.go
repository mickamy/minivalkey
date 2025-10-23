package store

import (
	"sync"
	"time"
)

// ValueType enumerates supported types (only string for MVP).
type ValueType int

const (
	TString ValueType = iota
)

// entry holds one key's payload & metadata.
// For simplicity, we keep a typed field per supported kind (s for string).
type entry struct {
	typ      ValueType
	s        string
	expireAt time.Time // zero => no expiry
}

// Store is a minimal in-memory KV with TTL support.
// Concurrency: RWMutex guards all access.
type Store struct {
	mu   sync.RWMutex
	data map[string]*entry
}

// New constructs an empty store.
func New() *Store {
	return &Store{data: make(map[string]*entry)}
}

// SetString sets key to string value with optional expiration.
func (st *Store) SetString(k, v string, expireAt time.Time) {
	st.mu.Lock()
	st.data[k] = &entry{typ: TString, s: v, expireAt: expireAt}
	st.mu.Unlock()
}

// GetString fetches string value if key exists and is not expired.
// Returns (value, true) if ok; otherwise ("", false).
func (st *Store) GetString(now time.Time, k string) (string, bool) {
	st.mu.RLock()
	e, ok := st.data[k]
	st.mu.RUnlock()
	if !ok {
		return "", false
	}
	// Lazy expiration on access
	if !e.expireAt.IsZero() && now.After(e.expireAt) {
		st.mu.Lock()
		delete(st.data, k)
		st.mu.Unlock()
		return "", false
	}
	if e.typ != TString {
		return "", false
	}
	return e.s, true
}

// Del deletes given keys and returns the number of removed entries.
func (st *Store) Del(keys ...string) int {
	st.mu.Lock()
	defer st.mu.Unlock()
	n := 0
	for _, k := range keys {
		if _, ok := st.data[k]; ok {
			delete(st.data, k)
			n++
		}
	}
	return n
}

// Exists returns count of keys that exist and are not expired at "now".
func (st *Store) Exists(now time.Time, keys ...string) int {
	st.mu.Lock()
	defer st.mu.Unlock()
	n := 0
	for _, k := range keys {
		e, ok := st.data[k]
		if !ok {
			continue
		}
		if !e.expireAt.IsZero() && now.After(e.expireAt) {
			delete(st.data, k)
			continue
		}
		n++
	}
	return n
}

// Expire sets a TTL in seconds for a key.
// sec < 0 removes expiration (persist).
// Returns false if key does not exist.
func (st *Store) Expire(now time.Time, k string, sec int64) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	e, ok := st.data[k]
	if !ok {
		return false
	}
	if sec < 0 {
		e.expireAt = time.Time{}
		return true
	}
	e.expireAt = now.Add(time.Duration(sec) * time.Second)
	return true
}

// TTL returns remaining time-to-live in seconds.
//
// Redis semantics:
//   - -2: key does not exist
//   - -1: key exists but has no associated expire
func (st *Store) TTL(now time.Time, k string) int64 {
	st.mu.RLock()
	e, ok := st.data[k]
	st.mu.RUnlock()
	if !ok {
		return -2
	}
	if e.expireAt.IsZero() {
		return -1
	}
	if now.After(e.expireAt) {
		// Ensure consistent view by removing expired entry.
		st.mu.Lock()
		delete(st.data, k)
		st.mu.Unlock()
		return -2
	}
	return int64(e.expireAt.Sub(now).Seconds())
}

// Stats returns simple keyspace stats at "now".
// keys: total keys
// expires: number of keys that have expiration set and are not yet expired at "now"
// avgTTLms: average TTL in milliseconds among keys that have expiration (>0); 0 if none.
func (st *Store) Stats(now time.Time) (keys int, expires int, avgTTLms int64) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var ttlSum time.Duration
	var ttlCount int

	for _, e := range st.data {
		// Skip already expired ones (lazy deletion will remove them later)
		if !e.expireAt.IsZero() && now.After(e.expireAt) {
			continue
		}
		keys++
		if !e.expireAt.IsZero() {
			expires++
			ttl := e.expireAt.Sub(now)
			if ttl > 0 {
				ttlSum += ttl
				ttlCount++
			}
		}
	}
	if ttlCount > 0 {
		avgTTLms = ttlSum.Milliseconds() / int64(ttlCount)
	}
	return
}

// CleanUpExpired scans entire map and removes expired entries.
// It's fine for test workloads (small maps). No fancy wheels required.
func (st *Store) CleanUpExpired(now time.Time) {
	st.mu.Lock()
	for k, e := range st.data {
		if !e.expireAt.IsZero() && now.After(e.expireAt) {
			delete(st.data, k)
		}
	}
	st.mu.Unlock()
}
