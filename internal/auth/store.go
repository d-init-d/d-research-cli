package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/d-init-d/d-research-cli/internal/paths"
	"github.com/zalando/go-keyring"
)

const envPrefix = "D_RESEARCH_"

type Entry struct {
	Ref        string `json:"ref"`
	Provider   string `json:"provider"`
	Source     string `json:"source"`
	Fingerprint string `json:"fingerprint"`
}

type Store struct {
	service string
}

func NewStore() *Store {
	return &Store{service: paths.KeyringService}
}

func (s *Store) Set(ref, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("secret cannot be empty")
	}
	return keyring.Set(s.service, ref, secret)
}

func (s *Store) Get(ref string) (string, error) {
	if v := s.fromEnv(ref); v != "" {
		return v, nil
	}
	v, err := keyring.Get(s.service, ref)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", fmt.Errorf("credential %s not found", ref)
		}
		return "", err
	}
	return v, nil
}

func (s *Store) Delete(ref string) error {
	if err := keyring.Delete(s.service, ref); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

func (s *Store) List(knownRefs []string) ([]Entry, error) {
	seen := map[string]bool{}
	var out []Entry
	for _, ref := range knownRefs {
		if seen[ref] {
			continue
		}
		seen[ref] = true
		source := "keyring"
		var fp string
		if v := s.fromEnv(ref); v != "" {
			source = "env"
			fp = fingerprint(v)
			out = append(out, Entry{Ref: ref, Provider: providerFromRef(ref), Source: source, Fingerprint: fp})
			continue
		}
		v, err := keyring.Get(s.service, ref)
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				continue
			}
			return nil, err
		}
		fp = fingerprint(v)
		out = append(out, Entry{Ref: ref, Provider: providerFromRef(ref), Source: source, Fingerprint: fp})
	}
	return out, nil
}

func (s *Store) Test(ref string) error {
	_, err := s.Get(ref)
	return err
}

func (s *Store) fromEnv(ref string) string {
	key := envPrefix + strings.ToUpper(strings.NewReplacer("/", "_", "-", "_").Replace(ref))
	return strings.TrimSpace(os.Getenv(key))
}

func providerFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ref
}

func fingerprint(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:4])
}

func CredentialRefFor(provider string) string {
	return "llm/" + provider
}

func SearchCredentialRef(providerID string) string {
	return "search/" + providerID
}