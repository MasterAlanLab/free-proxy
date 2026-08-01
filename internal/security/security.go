// Package security implements admin credential storage (scrypt), the session
// manager, and the auth service. The scrypt hash format is byte-compatible with
// the former Python implementation, so existing web-config.json hashes verify.
package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/store"
	"golang.org/x/crypto/scrypt"
)

// HashPassword returns a scrypt hash in the format scrypt$16384$8$1$salt$digest.
func HashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk, err := scrypt.Key([]byte(pw), salt, 1<<14, 8, 1, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("scrypt$16384$8$1$%s$%s",
		base64.URLEncoding.EncodeToString(salt),
		base64.URLEncoding.EncodeToString(dk)), nil
}

// VerifyPassword checks pw against an encoded scrypt hash.
func VerifyPassword(pw, encoded string) bool {
	p := strings.SplitN(encoded, "$", 6)
	if len(p) != 6 || p[0] != "scrypt" {
		return false
	}
	n, err1 := strconv.Atoi(p[1])
	r, err2 := strconv.Atoi(p[2])
	pp, err3 := strconv.Atoi(p[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	salt, err := base64.URLEncoding.DecodeString(p[4])
	if err != nil {
		return false
	}
	want, err := base64.URLEncoding.DecodeString(p[5])
	if err != nil {
		return false
	}
	got, err := scrypt.Key([]byte(pw), salt, n, r, pp, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

const credentialAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomCredential returns a random string whose first char is a letter and that
// contains at least one lower, one upper, and one digit.
func RandomCredential(length int) string {
	if length < 4 {
		length = 12
	}
	for {
		b := make([]byte, length)
		if _, err := rand.Read(b); err != nil {
			continue
		}
		out := make([]byte, length)
		for i, v := range b {
			out[i] = credentialAlphabet[int(v)%len(credentialAlphabet)]
		}
		s := string(out)
		if unicode.IsLetter(rune(s[0])) && strings.IndexFunc(s, unicode.IsLower) >= 0 &&
			strings.IndexFunc(s, unicode.IsUpper) >= 0 && strings.IndexFunc(s, unicode.IsDigit) >= 0 {
			return s
		}
	}
}

// AdminConfig is the database-backed admin/listener configuration. Host values
// are fixed listener constants; the ports and exposure flags are persisted.
type AdminConfig struct {
	Username            string
	PasswordHash        string
	SecretPath          string
	Host                string
	Port                int
	ProxyHost           string
	ProxyPort           int
	WebExternalAccess   *bool
	ProxyExternalAccess *bool
}

func (c AdminConfig) WebExternalAllowed() bool {
	return c.WebExternalAccess == nil || *c.WebExternalAccess
}

func (c AdminConfig) ProxyExternalAllowed() bool {
	return c.ProxyExternalAccess != nil && *c.ProxyExternalAccess
}

// AdminConfigStore persists management credentials in SQLite. web-config.json
// is read only for a one-time migration and is removed after the transaction.
type AdminConfigStore struct {
	cfg  *config.Config
	repo *store.AppSettingsRepository

	mu                sync.RWMutex
	config            AdminConfig
	bootstrapPassword string
}

// NewAdminConfigStore loads database settings, migrates legacy files, or creates
// random first-install credentials. Plaintext bootstrap passwords live only in
// this process and are never written to disk.
func NewAdminConfigStore(cfg *config.Config, repo *store.AppSettingsRepository) (*AdminConfigStore, error) {
	s := &AdminConfigStore{cfg: cfg, repo: repo}
	if err := s.loadOrCreate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *AdminConfigStore) Config() AdminConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *AdminConfigStore) Update(c AdminConfig) error {
	all, err := s.repo.Get(context.Background())
	if err != nil {
		return err
	}
	all.Admin.Username = c.Username
	all.Admin.PasswordHash = c.PasswordHash
	all.Admin.SecretPath = c.SecretPath
	all.Admin.WebPort = c.Port
	all.Admin.WebExternalAccess = c.WebExternalAllowed()
	all.Proxy.Port = c.ProxyPort
	all.Proxy.ExternalAccess = c.ProxyExternalAllowed()
	if err := s.repo.UpdateAdmin(context.Background(), all.Admin); err != nil {
		return err
	}
	if err := s.repo.UpdateProxy(context.Background(), all.Proxy); err != nil {
		return err
	}
	s.mu.Lock()
	s.config = c
	s.bootstrapPassword = ""
	s.mu.Unlock()
	return nil
}

func (s *AdminConfigStore) Rotate() (AdminConfig, string, error) {
	password := RandomCredential(12)
	hash, err := HashPassword(password)
	if err != nil {
		return AdminConfig{}, "", err
	}
	c := s.Config()
	c.Username, c.PasswordHash, c.SecretPath = RandomCredential(12), hash, RandomCredential(12)
	if err := s.Update(c); err != nil {
		return AdminConfig{}, "", err
	}
	s.mu.Lock()
	s.bootstrapPassword = password
	s.mu.Unlock()
	return c, password, nil
}

func (s *AdminConfigStore) SetExternalAccess(web, proxy bool) error {
	c := s.Config()
	c.WebExternalAccess, c.ProxyExternalAccess = &web, &proxy
	return s.Update(c)
}

func (s *AdminConfigStore) BootstrapPassword() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bootstrapPassword
}

func (s *AdminConfigStore) ClearBootstrapPassword() {
	s.mu.Lock()
	s.bootstrapPassword = ""
	s.mu.Unlock()
}

func (s *AdminConfigStore) loadOrCreate() error {
	ctx := context.Background()
	all, err := s.repo.Get(ctx)
	if err != nil {
		return err
	}
	legacyPath := filepath.Join(s.cfg.DataDir, "web-config.json")
	bootstrapPath := filepath.Join(s.cfg.DataDir, "initial-admin-password")
	legacy, legacyOK := readLegacyAdmin(legacyPath)
	if all.Admin.PasswordHash == "" && legacyOK {
		if legacy.PasswordHash == "" && legacy.PlaintextPassword != "" {
			legacy.PasswordHash, err = HashPassword(legacy.PlaintextPassword)
			if err != nil {
				return err
			}
		}
		all.Admin.Username = firstNonEmpty(legacy.Username, RandomCredential(12))
		all.Admin.PasswordHash = legacy.PasswordHash
		all.Admin.SecretPath = firstNonEmpty(legacy.SecretPath, RandomCredential(12))
		all.Admin.WebPort = legacy.Port
		if all.Admin.WebPort == 0 || all.Admin.WebPort == 8787 {
			all.Admin.WebPort = 39527
		}
		all.Admin.WebExternalAccess = legacy.WebExternalAllowed()
		if legacy.ProxyPort > 0 {
			all.Proxy.Port = legacy.ProxyPort
		}
		all.Proxy.ExternalAccess = legacy.ProxyExternalAllowed()
		if err = s.repo.UpdateAdmin(ctx, all.Admin); err != nil {
			return err
		}
		if err = s.repo.UpdateProxy(ctx, all.Proxy); err != nil {
			return err
		}
		if legacy.PlaintextPassword != "" {
			s.bootstrapPassword = legacy.PlaintextPassword
		}
	}
	if all.Admin.PasswordHash == "" {
		password := firstNonEmpty(s.cfg.AdminPassword, RandomCredential(12))
		hash, hashErr := HashPassword(password)
		if hashErr != nil {
			return hashErr
		}
		all.Admin.Username = firstNonEmpty(s.cfg.AdminUsername, RandomCredential(12))
		all.Admin.PasswordHash = hash
		all.Admin.SecretPath = firstNonEmpty(s.cfg.AdminSecretPath, RandomCredential(12))
		if all.Admin.WebPort == 0 || all.Admin.WebPort == 8787 {
			all.Admin.WebPort = 39527
		}
		if err = s.repo.UpdateAdmin(ctx, all.Admin); err != nil {
			return err
		}
		if s.cfg.AdminPassword == "" {
			s.bootstrapPassword = password
		}
	}
	// Preserve an old one-time password for the current install invocation only.
	if s.bootstrapPassword == "" {
		if data, readErr := os.ReadFile(bootstrapPath); readErr == nil {
			s.bootstrapPassword = strings.TrimSpace(string(data))
		}
	}
	_ = os.Remove(legacyPath)
	_ = os.Remove(bootstrapPath)
	web, proxy := all.Admin.WebExternalAccess, all.Proxy.ExternalAccess
	s.config = AdminConfig{
		Username: all.Admin.Username, PasswordHash: all.Admin.PasswordHash,
		SecretPath: all.Admin.SecretPath, Host: "0.0.0.0", Port: all.Admin.WebPort,
		ProxyHost: "0.0.0.0", ProxyPort: all.Proxy.Port,
		WebExternalAccess: &web, ProxyExternalAccess: &proxy,
	}
	return nil
}

type legacyAdminConfig struct {
	Username            string `json:"username"`
	PasswordHash        string `json:"password_hash"`
	PlaintextPassword   string `json:"password"`
	SecretPath          string `json:"secret_path"`
	Port                int    `json:"port"`
	ProxyPort           int    `json:"proxy_port"`
	WebExternalAccess   *bool  `json:"web_external_access"`
	ProxyExternalAccess *bool  `json:"proxy_external_access"`
}

func (c legacyAdminConfig) WebExternalAllowed() bool {
	return c.WebExternalAccess == nil || *c.WebExternalAccess
}
func (c legacyAdminConfig) ProxyExternalAllowed() bool {
	return c.ProxyExternalAccess != nil && *c.ProxyExternalAccess
}

func readLegacyAdmin(path string) (legacyAdminConfig, bool) {
	var c legacyAdminConfig
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &c) != nil {
		return c, false
	}
	return c, c.SecretPath != "" || c.PasswordHash != "" || c.PlaintextPassword != ""
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// SessionManager stores active session tokens in memory.
type SessionManager struct {
	ttl      time.Duration
	mu       sync.Mutex
	sessions map[string]time.Time
}

// NewSessionManager creates a SessionManager.
func NewSessionManager(ttl time.Duration) *SessionManager {
	return &SessionManager{ttl: ttl, sessions: map[string]time.Time{}}
}

// Create issues a new session token.
func (m *SessionManager) Create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := fmt.Sprintf("%x", b)
	m.mu.Lock()
	m.sessions[token] = time.Now().Add(m.ttl)
	m.mu.Unlock()
	return token, nil
}

// Valid reports whether a token is present and unexpired.
func (m *SessionManager) Valid(token string) bool {
	if token == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.sessions[token]
	if !ok || time.Now().After(exp) {
		delete(m.sessions, token)
		return false
	}
	return true
}

// Remove drops a token.
func (m *SessionManager) Remove(token string) {
	if token == "" {
		return
	}
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

// Clear drops all tokens.
func (m *SessionManager) Clear() {
	m.mu.Lock()
	m.sessions = map[string]time.Time{}
	m.mu.Unlock()
}

// AuthService verifies credentials and holds the store + sessions.
type AuthService struct {
	Cfg      *config.Config
	Store    *AdminConfigStore
	Sessions *SessionManager
}

// NewAuthService constructs an AuthService.
func NewAuthService(cfg *config.Config, store *AdminConfigStore, sessions *SessionManager) *AuthService {
	return &AuthService{Cfg: cfg, Store: store, Sessions: sessions}
}

// Verify checks a username/password against the stored config.
func (a *AuthService) Verify(username, password string) bool {
	c := a.Store.Config()
	ok := subtle.ConstantTimeCompare([]byte(username), []byte(c.Username)) == 1 &&
		VerifyPassword(password, c.PasswordHash)
	if ok {
		a.Store.ClearBootstrapPassword()
	}
	return ok
}
