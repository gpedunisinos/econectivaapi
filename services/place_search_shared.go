package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// One bind-mounted directory coordinates both API processes and survives restarts.
// The lock covers the upstream request, so cache hits and failures cannot bypass
// the start interval or allow overlapping requests from the other platform.
func sharedPlaceSearch(key string) (json.RawMessage, func([]byte), error) {
	dir := strings.TrimSpace(os.Getenv("PLACE_SEARCH_SHARED_DIR"))
	if dir == "" {
		return nil, func([]byte) {}, nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, nil, err
	}
	gate, err := os.OpenFile(filepath.Join(dir, "provider.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, nil, err
	}
	if err = unix.Flock(int(gate.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		gate.Close()
		return nil, nil, ErrPlaceBusy
	}
	unlock := func() { unix.Flock(int(gate.Fd()), unix.LOCK_UN); gate.Close() }
	sum := sha256.Sum256([]byte(key))
	path := filepath.Join(dir, hex.EncodeToString(sum[:])+".json")
	if info, e := os.Stat(path); e == nil && time.Since(info.ModTime()) < 24*time.Hour {
		if body, e := os.ReadFile(path); e == nil && json.Valid(body) {
			unlock()
			return body, func([]byte) {}, nil
		}
	}
	timestamp, e := os.ReadFile(filepath.Join(dir, "last-start"))
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		unlock()
		return nil, nil, e
	}
	if len(timestamp) > 0 {
		last, e := strconv.ParseInt(string(timestamp), 10, 64)
		if e != nil {
			unlock()
			return nil, nil, e
		}
		if time.Since(time.Unix(0, last)) < 1100*time.Millisecond {
			unlock()
			return nil, nil, ErrPlaceBusy
		}
	}
	if e = os.WriteFile(filepath.Join(dir, "last-start"), []byte(strconv.FormatInt(time.Now().UnixNano(), 10)), 0600); e != nil {
		unlock()
		return nil, nil, e
	}
	return nil, func(body []byte) {
		defer unlock()
		if len(body) == 0 {
			return
		}
		files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		if len(files) >= 512 {
			sort.Slice(files, func(i, j int) bool {
				a, _ := os.Stat(files[i])
				b, _ := os.Stat(files[j])
				return a.ModTime().Before(b.ModTime())
			})
			for len(files) >= 512 {
				os.Remove(files[0])
				files = files[1:]
			}
		}
		os.WriteFile(path, body, 0600)
	}, nil
}
