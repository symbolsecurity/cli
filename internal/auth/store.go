package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

const service = "symbol-cli"

var ErrNotLoggedIn = errors.New("not logged in")

type Credentials struct {
	APIKey       string `json:"api_key,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type OSKeyring struct{}

func (OSKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (OSKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (OSKeyring) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type MemoryKeyring struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMemoryKeyring() *MemoryKeyring {
	return &MemoryKeyring{data: map[string]string{}}
}

func (m *MemoryKeyring) key(service, user string) string {
	return service + "/" + user
}

func (m *MemoryKeyring) Get(service, user string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[m.key(service, user)]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (m *MemoryKeyring) Set(service, user, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[m.key(service, user)] = password
	return nil
}

func (m *MemoryKeyring) Delete(service, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := m.key(service, user)
	if _, ok := m.data[k]; !ok {
		return keyring.ErrNotFound
	}
	delete(m.data, k)
	return nil
}

type Store struct {
	Home    string
	Profile string
	Keyring Keyring
}

func NewStore(home, profile string, kr Keyring) *Store {
	if profile == "" {
		profile = "default"
	}
	if kr == nil {
		kr = OSKeyring{}
	}
	return &Store{Home: home, Profile: profile, Keyring: kr}
}

func (s *Store) user() string {
	return s.Profile
}

func (s *Store) filePath() string {
	name := "credentials.json"
	if s.Profile != "" && s.Profile != "default" {
		name = "credentials-" + s.Profile + ".json"
	}
	return filepath.Join(s.Home, ".config", "symbol", name)
}

func (s *Store) Load() (Credentials, error) {
	if raw, err := s.Keyring.Get(service, s.user()); err == nil && raw != "" {
		var c Credentials
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			return Credentials{}, err
		}
		if c.AccessToken != "" || c.APIKey != "" {
			return c, nil
		}
	}
	b, err := os.ReadFile(s.filePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, ErrNotLoggedIn
		}
		return Credentials{}, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return Credentials{}, err
	}
	if c.AccessToken == "" && c.APIKey == "" {
		return Credentials{}, ErrNotLoggedIn
	}
	return c, nil
}

func (s *Store) Save(c Credentials) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := s.Keyring.Set(service, s.user(), string(b)); err != nil {
		if mkErr := os.MkdirAll(filepath.Dir(s.filePath()), 0o700); mkErr != nil {
			return fmt.Errorf("keyring: %w; file: %v", err, mkErr)
		}
		if wErr := os.WriteFile(s.filePath(), b, 0o600); wErr != nil {
			return fmt.Errorf("keyring: %w; file: %v", err, wErr)
		}
		return nil
	}
	_ = os.Remove(s.filePath())
	return nil
}

func (s *Store) Clear() error {
	krErr := s.Keyring.Delete(service, s.user())
	fErr := os.Remove(s.filePath())
	if krErr != nil && !errors.Is(krErr, keyring.ErrNotFound) {
		if fErr != nil && !errors.Is(fErr, os.ErrNotExist) {
			return fmt.Errorf("keyring: %w; file: %v", krErr, fErr)
		}
		return krErr
	}
	if fErr != nil && !errors.Is(fErr, os.ErrNotExist) {
		return fErr
	}
	return nil
}

func (s *Store) UsingFile() bool {
	_, err := os.Stat(s.filePath())
	return err == nil
}

func (s *Store) WithLock(fn func() error) error {
	dir := filepath.Join(s.Home, ".config", "symbol")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	lockDir := filepath.Join(dir, s.Profile+".lock")
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := os.Mkdir(lockDir, 0o700)
		if err == nil {
			defer func() { _ = os.Remove(lockDir) }()
			return fn()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("credential lock timeout: %w", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
