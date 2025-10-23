package server

import (
	"strings"
	"time"

	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdSet(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountAtLeast(3)); err != nil {
		return w.WriteErrorAndFlush(err)
	}

	key := string(r.args[1])
	val := string(r.args[2])
	now := s.Now()

	opts := db.SetOptions{}
	returnOld := false
	for i := 3; i < len(r.args); i++ {
		opt := strings.ToUpper(string(r.args[i]))
		switch opt {
		case "NX":
			if opts.NX || opts.XX {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			opts.NX = true
		case "XX":
			if opts.XX || opts.NX {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			opts.XX = true
		case "KEEPTTL":
			if opts.KeepTTL || opts.HasExpire {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			opts.KeepTTL = true
		case "EX":
			if opts.HasExpire || opts.KeepTTL {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			i++
			if i >= len(r.args) {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			sec, ok := resp.ParseInt(r.args[i])
			if !ok {
				return w.WriteErrorAndFlush(ErrValueNotInteger)
			}
			if sec <= 0 {
				return w.WriteErrorAndFlush(ErrInvalidExpireTime)
			}
			opts.HasExpire = true
			opts.ExpireAt = now.Add(time.Duration(sec) * time.Second)
		case "PX":
			if opts.HasExpire || opts.KeepTTL {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			i++
			if i >= len(r.args) {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			ms, ok := resp.ParseInt(r.args[i])
			if !ok {
				return w.WriteErrorAndFlush(ErrValueNotInteger)
			}
			if ms <= 0 {
				return w.WriteErrorAndFlush(ErrInvalidExpireTime)
			}
			opts.HasExpire = true
			opts.ExpireAt = now.Add(time.Duration(ms) * time.Millisecond)
		case "EXAT":
			if opts.HasExpire || opts.KeepTTL {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			i++
			if i >= len(r.args) {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			sec, ok := resp.ParseInt(r.args[i])
			if !ok {
				return w.WriteErrorAndFlush(ErrValueNotInteger)
			}
			if sec <= 0 {
				return w.WriteErrorAndFlush(ErrInvalidExpireTime)
			}
			opts.HasExpire = true
			opts.ExpireAt = time.Unix(sec, 0)
		case "PXAT":
			if opts.HasExpire || opts.KeepTTL {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			i++
			if i >= len(r.args) {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			ms, ok := resp.ParseInt(r.args[i])
			if !ok {
				return w.WriteErrorAndFlush(ErrValueNotInteger)
			}
			if ms <= 0 {
				return w.WriteErrorAndFlush(ErrInvalidExpireTime)
			}
			opts.HasExpire = true
			opts.ExpireAt = time.UnixMilli(ms)
		case "GET":
			if returnOld {
				return w.WriteErrorAndFlush(ErrSyntax)
			}
			returnOld = true
		default:
			return w.WriteErrorAndFlush(ErrSyntax)
		}
	}

	stored, prev, prevExists := s.db(r.session).SetStringWithOptions(now, key, val, opts)
	if !stored {
		if err := w.WriteNull(); err != nil {
			return err
		}
		return nil
	}

	if returnOld {
		if prevExists {
			if err := w.WriteBulk([]byte(prev)); err != nil {
				return err
			}
		} else {
			if err := w.WriteNull(); err != nil {
				return err
			}
		}
		return nil
	}

	if err := w.WriteString("OK"); err != nil {
		return err
	}

	return nil
}
