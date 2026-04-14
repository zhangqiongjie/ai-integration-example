package provider

import (
	"context"
	"fmt"
	"sync"

	"voice-model/internal/retry"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]TextProvider
	active    string
	voice     VoiceProvider
	retryCfg  retry.Config
}

func NewRegistry(retryCfg retry.Config) *Registry {
	return &Registry{
		providers: make(map[string]TextProvider),
		retryCfg:  retryCfg,
	}
}

func (r *Registry) Register(p TextProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
	if r.active == "" {
		r.active = p.Name()
	}
}

func (r *Registry) SetVoice(v VoiceProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.voice = v
}

func (r *Registry) Voice() VoiceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.voice
}

func (r *Registry) Active() TextProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[r.active]
}

func (r *Registry) ActiveName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

func (r *Registry) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[name]; !ok {
		return fmt.Errorf("provider %q not registered", name)
	}
	r.active = name
	return nil
}

type ProviderInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

func (r *Registry) List() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ProviderInfo, 0, len(r.providers))
	for name := range r.providers {
		list = append(list, ProviderInfo{Name: name, Available: true})
	}
	return list
}

func (r *Registry) SendMessage(ctx context.Context, messages []Message, onChunk func(string)) (string, error) {
	p := r.Active()
	if p == nil {
		return "", fmt.Errorf("no active provider")
	}

	var result string
	err := retry.Do(ctx, r.retryCfg, func() error {
		var sendErr error
		result, sendErr = p.SendMessage(ctx, messages, onChunk)
		return sendErr
	})
	return result, err
}
