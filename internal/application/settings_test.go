package application

import (
	"termo-sync/internal/credentials"
	"termo-sync/internal/media"
	"termo-sync/internal/providers"
	"testing"
)

type memoryStore map[string]string

func (m memoryStore) Get(p string) (string, error) {
	if k := m[p]; k != "" {
		return k, nil
	}
	return "", credentials.ErrMissing
}
func (m memoryStore) Set(p, k string) error { m[p] = k; return nil }
func (m memoryStore) Delete(p string) error { delete(m, p); return nil }
func TestSettingsDoNotExposeKeys(t *testing.T) {
	m := memoryStore{}
	s := NewService(m, providers.New(), media.Tools{})
	if err := s.SaveKey("assemblyai", "  test-key  "); err != nil {
		t.Fatal(err)
	}
	if m["assemblyai"] != "test-key" || !s.Settings().AssemblyAIConfigured {
		t.Fatal("key was not stored")
	}
	if s.Settings().DeepSeekConfigured {
		t.Fatal("wrong provider")
	}
	if err := s.SaveKey("deepseek", "key\ninvalid"); err == nil {
		t.Fatal("header injection accepted")
	}
	if err := s.SetModel("arbitrary-model"); err == nil {
		t.Fatal("unknown model accepted")
	}
	if err := s.SetModel("deepseek-v4-pro"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveKey("assemblyai"); err != nil {
		t.Fatal(err)
	}
	if s.Settings().AssemblyAIConfigured {
		t.Fatal("key not removed")
	}
	s.busy = true
	if err := s.SaveKey("assemblyai", "key"); err == nil {
		t.Fatal("credential changed during processing")
	}
}
