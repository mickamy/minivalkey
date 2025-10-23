package db

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

// DB is a minimal in-memory KV with TTL support.
// Concurrency: RWMutex guards all access.
type DB struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

// New constructs an empty db.
func New() *DB {
	return &DB{entries: make(map[string]*entry)}
}

// SetOptions tweaks SetStringWithOptions behavior.
type SetOptions struct {
	NX        bool // Only set if key does not exist
	XX        bool // Only set if key exists
	KeepTTL   bool // Retain existing TTL if any
	ExpireAt  time.Time
	HasExpire bool
}

// SetString sets key to string value with optional expiration.
func (db *DB) SetString(k, v string, expireAt time.Time) {
	db.mu.Lock()
	db.entries[k] = &entry{typ: TString, s: v, expireAt: expireAt}
	db.mu.Unlock()
}

// SetStringWithOptions sets key to value honouring NX/XX/KEEPTTL and optional expiry.
// Returns true if the value was stored, false if the preconditions failed (e.g. NX with existing key).
func (db *DB) SetStringWithOptions(now time.Time, k, v string, opts SetOptions) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	e, exists := db.entries[k]
	// Drop stale entries so existence checks match read paths.
	if exists && !e.expireAt.IsZero() && now.After(e.expireAt) {
		delete(db.entries, k)
		exists = false
		e = nil
	}

	if opts.NX && exists {
		return false
	}
	if opts.XX && !exists {
		return false
	}

	expireAt := time.Time{}
	if opts.KeepTTL && exists {
		expireAt = e.expireAt
	}
	if opts.HasExpire {
		expireAt = opts.ExpireAt
	}

	db.entries[k] = &entry{typ: TString, s: v, expireAt: expireAt}
	return true
}

// GetString fetches string value if key exists and is not expired.
// Returns (value, true) if ok; otherwise ("", false).
func (db *DB) GetString(now time.Time, k string) (string, bool) {
	db.mu.RLock()
	e, ok := db.entries[k]
	db.mu.RUnlock()
	if !ok {
		return "", false
	}
	// Lazy expiration on access
	if !e.expireAt.IsZero() && now.After(e.expireAt) {
		db.mu.Lock()
		delete(db.entries, k)
		db.mu.Unlock()
		return "", false
	}
	if e.typ != TString {
		return "", false
	}
	return e.s, true
}

// Del deletes given keys and returns the number of removed entries.
func (db *DB) Del(keys ...string) int {
	db.mu.Lock()
	defer db.mu.Unlock()
	n := 0
	for _, k := range keys {
		if _, ok := db.entries[k]; ok {
			delete(db.entries, k)
			n++
		}
	}
	return n
}

// Exists returns count of keys that exist and are not expired at "now".
func (db *DB) Exists(now time.Time, keys ...string) int {
	db.mu.Lock()
	defer db.mu.Unlock()
	n := 0
	for _, k := range keys {
		e, ok := db.entries[k]
		if !ok {
			continue
		}
		if !e.expireAt.IsZero() && now.After(e.expireAt) {
			delete(db.entries, k)
			continue
		}
		n++
	}
	return n
}

// Expire sets a TTL in seconds for a key.
// sec < 0 removes expiration (persist).
// Returns false if key does not exist.
func (db *DB) Expire(now time.Time, k string, sec int64) bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	e, ok := db.entries[k]
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
func (db *DB) TTL(now time.Time, k string) int64 {
	db.mu.RLock()
	e, ok := db.entries[k]
	db.mu.RUnlock()
	if !ok {
		return -2
	}
	if e.expireAt.IsZero() {
		return -1
	}
	if now.After(e.expireAt) {
		// Ensure consistent view by removing expired entry.
		db.mu.Lock()
		delete(db.entries, k)
		db.mu.Unlock()
		return -2
	}
	return int64(e.expireAt.Sub(now).Seconds())
}

// Stats returns simple keyspace stats at "now".
// keys: total keys
// expires: number of keys that have expiration set and are not yet expired at "now"
// avgTTLms: average TTL in milliseconds among keys that have expiration (>0); 0 if none.
func (db *DB) Stats(now time.Time) (keys int, expires int, avgTTLms int64) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var ttlSum time.Duration
	var ttlCount int

	for _, e := range db.entries {
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
func (db *DB) CleanUpExpired(now time.Time) {
	db.mu.Lock()
	for k, e := range db.entries {
		if !e.expireAt.IsZero() && now.After(e.expireAt) {
			delete(db.entries, k)
		}
	}
	db.mu.Unlock()
}
