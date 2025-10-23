package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
)

var (
	ErrInfoUnknownSection = fmt.Errorf("ERR unknown section")
)

func (s *Server) cmdInfo(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	// RESP2: INFO [section]
	// We support sections: "server", "memory", "keyspace", plus "all"/"default".
	// Unknown sections -> error (to match Redis/Valkey behavior).
	section := "default"
	if len(args) == 2 {
		section = strings.ToLower(string(args[1]))
	}
	// Build content based on requested section.
	now := s.Now()
	txt, ok := buildInfo(section, now, s.db, s.uptimeSeconds(now))
	if !ok {
		return w.WriteErrorAndFlush(ErrInfoUnknownSection)
	}
	if err := w.WriteBulk([]byte(txt)); err != nil {
		return err
	}

	return nil
}

// buildInfo builds an INFO string for a given section.
// Returns (text, true) if section is supported; ("", false) otherwise.
func buildInfo(section string, now time.Time, st *db.DB, uptimeSec int64) (string, bool) {
	switch section {
	case "all", "default":
		var b strings.Builder
		b.WriteString(infoServer(now, uptimeSec))
		b.WriteString(infoMemory(now, st))
		b.WriteString(infoKeyspace(now, st))
		return b.String(), true
	case "server":
		return infoServer(now, uptimeSec), true
	case "memory":
		return infoMemory(now, st), true
	case "keyspace":
		return infoKeyspace(now, st), true
	case "replication":
		return infoReplication(), true
	default:
		fmt.Println("unknown info section:", section)
		return "", false
	}
}

func infoServer(now time.Time, uptimeSec int64) string {
	var b strings.Builder
	b.WriteString("# Server\r\n")
	// server: string identifier (we advertise "valkey" for compatibility)
	b.WriteString("server:valkey\r\n")
	// version: library version; keep "0.0.0" for now, or wire to a const
	b.WriteString("version:0.0.0\r\n")
	// proto: we speak RESP2
	b.WriteString("proto:2\r\n")
	// process_id: arbitrary positive id (we don't fork, so constant is fine)
	b.WriteString("process_id:1\r\n")
	// uptime_in_seconds: based on simulated clock
	b.WriteString("uptime_in_seconds:")
	b.WriteString(strconv.FormatInt(uptimeSec, 10))
	b.WriteString("\r\n")
	// mode/role: single node master-like
	b.WriteString("mode:standalone\r\n")
	b.WriteString("role:master\r\n")
	// time_now: unix seconds (simulated clock)
	b.WriteString("time_now:")
	b.WriteString(strconv.FormatInt(now.Unix(), 10))
	b.WriteString("\r\n\r\n")
	return b.String()
}

func infoMemory(now time.Time, st *db.DB) string {
	keys, expires, _ := st.Stats(now)
	var b strings.Builder
	b.WriteString("# Memory\r\n")
	// used_memory: we don't track bytes; expose number of keys as a hint
	b.WriteString("used_memory_keys:")
	b.WriteString(strconv.Itoa(keys))
	b.WriteString("\r\n")
	// expires: number of keys with TTL
	b.WriteString("expires:")
	b.WriteString(strconv.Itoa(expires))
	b.WriteString("\r\n\r\n")
	return b.String()
}

func infoKeyspace(now time.Time, st *db.DB) string {
	keys, expires, avgTTLms := st.Stats(now)
	var b strings.Builder
	b.WriteString("# Keyspace\r\n")
	// Only emit db0 if there are any keys (mimic Redis behavior)
	if keys > 0 {
		// format: db0:keys=<int>,expires=<int>,avg_ttl=<milliseconds>
		b.WriteString("db0:")
		b.WriteString("keys=")
		b.WriteString(strconv.Itoa(keys))
		b.WriteString(",expires=")
		b.WriteString(strconv.Itoa(expires))
		b.WriteString(",avg_ttl=")
		b.WriteString(strconv.FormatInt(avgTTLms, 10))
		b.WriteString("\r\n")
	}
	b.WriteString("\r\n")
	return b.String()
}

func infoReplication() string {
	// Minimal, master-only, no backlog. Enough for clients probing replication.
	var b strings.Builder
	b.WriteString("# Replication\r\n")
	b.WriteString("role:master\r\n")
	b.WriteString("connected_slaves:0\r\n")
	// 40 hex chars, dummy but valid-shaped replid values
	b.WriteString("master_replid:0000000000000000000000000000000000000000\r\n")
	b.WriteString("master_replid2:0000000000000000000000000000000000000000\r\n")
	b.WriteString("master_repl_offset:0\r\n")
	b.WriteString("second_repl_offset:-1\r\n")
	b.WriteString("repl_backlog_active:0\r\n\r\n")
	return b.String()
}
