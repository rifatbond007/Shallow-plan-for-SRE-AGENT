package storage

import (
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/analysis"
)

type Store struct {
	cache *Cache
}

func NewStore(maxSize int, ttl time.Duration) *Store {
	return &Store{
		cache: NewCache(maxSize, ttl),
	}
}

func (s *Store) Get(id string) (*analysis.AnalysisResult, bool) {
	v, ok := s.cache.Get(id)
	if !ok {
		return nil, false
	}
	return v.(*analysis.AnalysisResult), true
}

func (s *Store) Set(id string, result *analysis.AnalysisResult) {
	s.cache.Set(id, result)
}
