package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"

	"github.com/mickamy/minivalkey"
)

// E2E tests using valkey-go client.
// Verifies PING, SET, GET, DEL, EXPIRE, TTL and simulated time.
func TestBasicCommandsAndTTL_WithValkeyGo(t *testing.T) {
	// Boot in-memory server
	s, err := minivalkey.Run()
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	// Build a valkey-go client (standalone)
	client := newValkeyClient(t, s)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	ctx := t.Context()

	// --- PING (no payload) ---
	resp := client.Do(ctx, client.B().Ping().Build())
	require.NoError(t, resp.Error())
	str, err := resp.ToString()
	require.NoError(t, err)
	assert.Equal(t, "PONG", str)

	// --- PING (with payload) ---
	resp = client.Do(ctx, client.B().Ping().Message("hello").Build())
	require.NoError(t, resp.Error())
	str, err = resp.ToString()
	require.NoError(t, err)
	assert.Equal(t, "hello", str)

	// --- SET / GET ---
	require.NoError(t, client.Do(ctx, client.B().Set().Key("k").Value("v").Build()).Error())
	got := client.Do(ctx, client.B().Get().Key("k").Build())
	require.NoError(t, got.Error())
	sv, err := got.ToString()
	require.NoError(t, err)
	assert.Equal(t, "v", sv)

	// --- DEL ---
	del := client.Do(ctx, client.B().Del().Key("k").Build())
	require.NoError(t, del.Error())
	n, err := del.AsInt64()
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	// Deleted key should be absent
	gone := client.Do(ctx, client.B().Get().Key("k").Build())
	require.True(t, valkey.IsValkeyNil(gone.Error()))
	_, err = gone.ToString()
	assert.Error(t, err, "expect conversion error for nil bulk")

	// --- EXPIRE / TTL ---
	require.NoError(t, client.Do(ctx, client.B().Set().Key("ttl").Value("x").Build()).Error())
	require.NoError(t, client.Do(ctx, client.B().Expire().Key("ttl").Seconds(5).Build()).Error())

	ttl := client.Do(ctx, client.B().Ttl().Key("ttl").Build())
	require.NoError(t, ttl.Error())
	dur, err := ttl.AsInt64()
	require.NoError(t, err)
	assert.Greater(t, dur, int64(0), "TTL should be positive")

	// Fast-forward virtual time to expire
	s.FastForward(6 * time.Second)

	// Key should be expired
	exp := client.Do(ctx, client.B().Get().Key("ttl").Build())
	require.True(t, valkey.IsValkeyNil(exp.Error()))
	_, err = exp.ToString()
	assert.Error(t, err, "expired key should be nil bulk")

	// TTL should be -2 (no such key)
	ttl2 := client.Do(ctx, client.B().Ttl().Key("ttl").Build())
	require.NoError(t, ttl2.Error())
	dur2, err := ttl2.AsInt64()
	require.NoError(t, err)
	assert.Equal(t, int64(-2), dur2)
}

func TestTTL_NoExpire_And_NonExisting_WithValkeyGo(t *testing.T) {
	s, err := minivalkey.Run()
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	client := newValkeyClient(t, s)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()

	// TTL on non-existing key -> -2
	ttl := client.Do(ctx, client.B().Ttl().Key("nope").Build())
	require.NoError(t, ttl.Error())
	n, err := ttl.AsInt64()
	require.NoError(t, err)
	assert.Equal(t, int64(-2), n)

	// Key without expire -> TTL = -1
	require.NoError(t, client.Do(ctx, client.B().Set().Key("plain").Value("v").Build()).Error())
	ttl = client.Do(ctx, client.B().Ttl().Key("plain").Build())
	require.NoError(t, ttl.Error())
	n, err = ttl.AsInt64()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), n)
}

func newValkeyClient(t *testing.T, s *minivalkey.Server) valkey.Client {
	t.Helper()
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:           []string{s.Addr()},
		DisableCache:          true, // Disable caching for test determinism
		DisableAutoPipelining: true, // DisableAutoPipelining can be turned on for determinism
	})
	if err != nil {
		t.Fatalf("failed to create valkey client: %v", err)
	}
	return client
}
