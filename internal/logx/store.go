package logx

import (
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type Store struct {
	mu      sync.Mutex
	entries []Entry
	limit   int
	subs    map[chan Entry]struct{}
}

func NewStore() *Store {
	return &Store{
		entries: make([]Entry, 0, 128),
		limit:   200,
		subs:    make(map[chan Entry]struct{}),
	}
}

func (s *Store) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (s *Store) Fire(entry *logrus.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.limit > 0 && len(s.entries) >= s.limit {
		copy(s.entries, s.entries[1:])
		s.entries = s.entries[:s.limit-1]
	}

	s.entries = append(s.entries, Entry{
		Time:    entry.Time,
		Level:   strings.ToUpper(entry.Level.String()),
		Message: entry.Message,
	})

	for ch := range s.subs {
		select {
		case ch <- s.entries[len(s.entries)-1]:
		default:
		}
	}
	return nil
}

func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *Store) Recent(limit int) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit >= len(s.entries) {
		out := make([]Entry, len(s.entries))
		copy(out, s.entries)
		return out
	}

	start := len(s.entries) - limit
	out := make([]Entry, limit)
	copy(out, s.entries[start:])
	return out
}

func (s *Store) Subscribe(buffer int) (<-chan Entry, func()) {
	if buffer <= 0 {
		buffer = 1
	}

	ch := make(chan Entry, buffer)

	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()

	var once sync.Once

	return ch, func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			if _, ok := s.subs[ch]; !ok {
				return
			}

			delete(s.subs, ch)
			close(ch)
		})
	}
}
