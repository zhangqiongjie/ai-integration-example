package conversation

import "sync"

type Store struct {
	mu    sync.RWMutex
	convs map[string]*Conversation
}

func NewStore() *Store {
	return &Store{
		convs: make(map[string]*Conversation),
	}
}

func (s *Store) Create(providerName, systemPrompt string, maxMessages int) *Conversation {
	conv := New(providerName, systemPrompt, maxMessages)
	s.mu.Lock()
	s.convs[conv.ID] = conv
	s.mu.Unlock()
	return conv
}

func (s *Store) Get(id string) (*Conversation, bool) {
	s.mu.RLock()
	conv, ok := s.convs[id]
	s.mu.RUnlock()
	return conv, ok
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.convs, id)
	s.mu.Unlock()
}
