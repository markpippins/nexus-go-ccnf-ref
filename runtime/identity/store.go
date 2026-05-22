package identity

import "sync"

type Store interface {
	Get(sig SemanticSignature) (OriginSeed, bool)
	Insert(sig SemanticSignature, seed OriginSeed) error
}

type memoryStore struct {
	mu   sync.Mutex
	data map[SemanticSignature]OriginSeed
}

func NewMemoryStore() Store {
	return &memoryStore{data: make(map[SemanticSignature]OriginSeed)}
}

func (s *memoryStore) Get(sig SemanticSignature) (OriginSeed, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seed, ok := s.data[sig]
	return seed, ok
}

func (s *memoryStore) Insert(sig SemanticSignature, seed OriginSeed) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[sig]; ok {
		return ErrAlreadyExists
	}
	s.data[sig] = seed
	return nil
}
