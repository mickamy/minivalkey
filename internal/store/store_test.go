package store

import (
	"testing"
	"time"
)

func TestStore_GetString(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0)

	tcs := []struct {
		name    string
		arrange func(*Store)
		key     string
		want    string
		wantOK  bool
	}{
		{
			name: "returns stored value",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", time.Time{})
			},
			key:    "foo",
			want:   "bar",
			wantOK: true,
		},
		{
			name: "removes expired key on access",
			arrange: func(st *Store) {
				st.SetString("gone", "x", now.Add(-time.Second))
			},
			key:    "gone",
			wantOK: false,
		},
		{
			name: "returns false for missing key",
			key:  "missing",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			got, ok := st.GetString(now, tc.key)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("GetString(%q) = (%q,%v); want (%q,%v)", tc.key, got, ok, tc.want, tc.wantOK)
			}
			if !tc.wantOK && tc.arrange != nil {
				if _, exists := st.data[tc.key]; exists {
					t.Fatalf("expected key %q to be removed after access", tc.key)
				}
			}
		})
	}
}

func TestStore_Del(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name    string
		arrange func(*Store)
		keys    []string
		want    int
	}{
		{
			name: "removes existing keys",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", time.Time{})
				st.SetString("baz", "qux", time.Time{})
			},
			keys: []string{"foo", "baz", "missing"},
			want: 2,
		},
		{
			name: "returns zero when keys are missing",
			keys: []string{"nope"},
			want: 0,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			if got := st.Del(tc.keys...); got != tc.want {
				t.Fatalf("Del(%v) = %d; want %d", tc.keys, got, tc.want)
			}
			for _, k := range tc.keys {
				if _, ok := st.GetString(time.Time{}, k); ok {
					t.Fatalf("expected key %q to be deleted", k)
				}
			}
		})
	}
}

func TestStore_Exists(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0)

	tcs := []struct {
		name    string
		arrange func(*Store)
		keys    []string
		want    int
	}{
		{
			name: "counts present keys",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", time.Time{})
				st.SetString("baz", "qux", now.Add(time.Minute))
			},
			keys: []string{"foo", "baz", "missing"},
			want: 2,
		},
		{
			name: "skips expired keys",
			arrange: func(st *Store) {
				st.SetString("old", "x", now.Add(-time.Second))
				st.SetString("fresh", "y", time.Time{})
			},
			keys: []string{"old", "fresh"},
			want: 1,
		},
		{
			name: "returns zero when store is empty",
			keys: []string{"foo"},
			want: 0,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			if got := st.Exists(now, tc.keys...); got != tc.want {
				t.Fatalf("Exists(%v) = %d; want %d", tc.keys, got, tc.want)
			}
		})
	}
}

func TestStore_Expire(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0)

	tcs := []struct {
		name    string
		arrange func(*Store)
		key     string
		sec     int64
		want    bool
		check   func(*testing.T, *Store)
	}{
		{
			name: "sets expiration when key exists",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", time.Time{})
			},
			key:  "foo",
			sec:  10,
			want: true,
			check: func(t *testing.T, st *Store) {
				if ttl := st.TTL(now.Add(5*time.Second), "foo"); ttl != 5 {
					t.Fatalf("expected ttl 5, got %d", ttl)
				}
			},
		},
		{
			name: "removes expiration when seconds negative",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", now.Add(10*time.Second))
			},
			key:  "foo",
			sec:  -1,
			want: true,
			check: func(t *testing.T, st *Store) {
				if ttl := st.TTL(now, "foo"); ttl != -1 {
					t.Fatalf("expected ttl -1, got %d", ttl)
				}
			},
		},
		{
			name: "returns false when key missing",
			key:  "missing",
			sec:  5,
			want: false,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			if got := st.Expire(now, tc.key, tc.sec); got != tc.want {
				t.Fatalf("Expire(%q,%d) = %v; want %v", tc.key, tc.sec, got, tc.want)
			}
			if tc.check != nil {
				tc.check(t, st)
			}
		})
	}
}

func TestStore_TTL(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0)

	tcs := []struct {
		name    string
		arrange func(*Store)
		key     string
		want    int64
	}{
		{
			name: "returns remaining seconds when key has expiry",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", now.Add(10*time.Second))
			},
			key:  "foo",
			want: 10,
		},
		{
			name: "returns minus one when key has no expiry",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", time.Time{})
			},
			key:  "foo",
			want: -1,
		},
		{
			name: "returns minus two when key missing",
			key:  "missing",
			want: -2,
		},
		{
			name: "removes expired key and returns minus two",
			arrange: func(st *Store) {
				st.SetString("foo", "bar", now.Add(-time.Second))
			},
			key:  "foo",
			want: -2,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			if got := st.TTL(now, tc.key); got != tc.want {
				t.Fatalf("TTL(%q) = %d; want %d", tc.key, got, tc.want)
			}
			if tc.want == -2 {
				if _, exists := st.data[tc.key]; exists {
					t.Fatalf("expected key %q to be removed after TTL check", tc.key)
				}
			}
		})
	}
}

func TestStore_Stats(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0)

	st := New()
	st.SetString("foo", "bar", time.Time{})
	st.SetString("baz", "qux", now.Add(10*time.Second))
	st.SetString("expired", "x", now.Add(-time.Second))

	keys, expires, avgTTL := st.Stats(now)

	if keys != 2 {
		t.Fatalf("Stats keys = %d; want 2", keys)
	}
	if expires != 1 {
		t.Fatalf("Stats expires = %d; want 1", expires)
	}
	if avgTTL != 10000 {
		t.Fatalf("Stats avgTTLms = %d; want 10000", avgTTL)
	}
}

func TestStore_CleanUpExpired(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0)

	st := New()
	st.SetString("fresh", "1", now.Add(time.Minute))
	st.SetString("stale", "2", now.Add(-time.Second))

	st.CleanUpExpired(now)

	if _, ok := st.GetString(now, "stale"); ok {
		t.Fatalf("expected stale key to be removed")
	}
	if _, ok := st.GetString(now, "fresh"); !ok {
		t.Fatalf("expected fresh key to remain")
	}
}
