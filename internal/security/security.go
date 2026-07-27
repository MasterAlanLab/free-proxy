// Package security implements admin credential storage (scrypt), the session
// manager, and the auth service. The scrypt hash format is byte-compatible with
// the former Python implementation, so existing web-config.json hashes verify.
package security

import (
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

// AdminConfig is the persisted admin/listener configuration.
type AdminConfig struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	SecretPath   string `json:"secret_path"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ProxyHost    string `json:"proxy_host"`
	ProxyPort    int    `json:"proxy_port"`
	// Network-exposure toggles. Listeners always bind 0.0.0.0; these gate whether
	// non-loopback clients are actually served, and can be flipped at runtime from
	// the admin UI. nil means "unset" and falls back to the safe default below.
	WebExternalAccess   *bool `json:"web_external_access,omitempty"`
	ProxyExternalAccess *bool `json:"proxy_external_access,omitempty"`
}

// WebExternalAllowed reports whether the web admin serves non-loopback clients.
// Default true: the admin UI is already protected by login + a secret path.
func (c AdminConfig) WebExternalAllowed() bool {
	return c.WebExternalAccess == nil || *c.WebExternalAccess
}

// ProxyExternalAllowed reports whether the proxy serves non-loopback clients.
// Default false: avoids accidentally exposing an open proxy to the internet.
func (c AdminConfig) ProxyExternalAllowed() bool {
	return c.ProxyExternalAccess != nil && *c.ProxyExternalAccess
}

// AdminConfigStore loads/persists the admin config and one-time bootstrap password.
type AdminConfigStore struct {
	cfg           *config.Config
	path          string
	bootstrapPath string

	mu     sync.RWMutex
	config AdminConfig
}

// NewAdminConfigStore loads or creates the admin config.
func NewAdminConfigStore(cfg *config.Config) (*AdminConfigStore, error) {
	s := &AdminConfigStore{
		cfg:           cfg,
		path:          filepath.Join(cfg.DataDir, "web-config.json"),
		bootstrapPath: filepath.Join(cfg.DataDir, "initial-admin-password"),
	}
	c, err := s.loadOrCreate()
	if err != nil {
		return nil, err
	}
	s.config = c
	return s, nil
}

// Config returns a copy of the current config.
func (s *AdminConfigStore) Config() AdminConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// Update persists a new config and clears the bootstrap password.
func (s *AdminConfigStore) Update(c AdminConfig) error {
	if err := s.write(c); err != nil {
		return err
	}
	s.mu.Lock()
	s.config = c
	s.mu.Unlock()
	s.ClearBootstrapPassword()
	return nil
}

// SetExternalAccess flips the network-exposure toggles at runtime (persisted +
// in-memory) without touching credentials or the bootstrap password.
func (s *AdminConfigStore) SetExternalAccess(web, proxy bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.config
	c.WebExternalAccess = &web
	c.ProxyExternalAccess = &proxy
	if err := s.write(c); err != nil {
		return err
	}
	s.config = c
	return nil
}

// BootstrapPassword returns the one-time password, or "" if none.
func (s *AdminConfigStore) BootstrapPassword() string {
	data, err := os.ReadFile(s.bootstrapPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ClearBootstrapPassword deletes the one-time password file.
func (s *AdminConfigStore) ClearBootstrapPassword() {
	_ = os.Remove(s.bootstrapPath)
}

func (s *AdminConfigStore) loadOrCreate() (AdminConfig, error) {
	if data, err := os.ReadFile(s.path); err == nil {
		var raw map[string]any
		if json.Unmarshal(data, &raw) == nil {
			if pw, ok := raw["password"].(string); ok && pw != "" {
				if _, hashed := raw["password_hash"].(string); !hashed {
					c, err := s.migrate(raw, pw)
					if err == nil {
						return c, s.write(c)
					}
				}
			}
			var c AdminConfig
			if json.Unmarshal(data, &c) == nil && c.SecretPath != "" {
				return c, nil
			}
		}
	}
	password := s.cfg.AdminPassword
	if password == "" {
		password = RandomCredential(12)
	}
	hash, err := HashPassword(password)
	if err != nil {
		return AdminConfig{}, err
	}
	c := AdminConfig{
		Username:     firstNonEmpty(s.cfg.AdminUsername, RandomCredential(12)),
		PasswordHash: hash,
		SecretPath:   firstNonEmpty(s.cfg.AdminSecretPath, RandomCredential(12)),
		Host:         s.cfg.WebHost,
		Port:         s.cfg.WebPort,
		ProxyHost:    s.cfg.ProxyHost,
		ProxyPort:    s.cfg.ProxyPort,
	}
	if err := s.write(c); err != nil {
		return AdminConfig{}, err
	}
	if s.cfg.AdminPassword == "" {
		s.writeBootstrap(password)
	}
	return c, nil
}

func (s *AdminConfigStore) migrate(raw map[string]any, plaintext string) (AdminConfig, error) {
	hash, err := HashPassword(plaintext)
	if err != nil {
		return AdminConfig{}, err
	}
	str := func(k, def string) string {
		if v, ok := raw[k].(string); ok && v != "" {
			return v
		}
		return def
	}
	num := func(k string, def int) int {
		if v, ok := raw[k].(float64); ok {
			return int(v)
		}
		return def
	}
	return AdminConfig{
		Username:     str("username", RandomCredential(12)),
		PasswordHash: hash,
		SecretPath:   str("secret_path", RandomCredential(12)),
		Host:         str("host", s.cfg.WebHost),
		Port:         num("port", s.cfg.WebPort),
		ProxyHost:    str("proxy_host", s.cfg.ProxyHost),
		ProxyPort:    num("proxy_port", s.cfg.ProxyPort),
	}, nil
}

func (s *AdminConfigStore) write(c AdminConfig) error {
	if err := s.cfg.EnsureDirectories(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	_ = os.Chmod(s.path, 0o600)
	return nil
}

func (s *AdminConfigStore) writeBootstrap(password string) {
	_ = os.WriteFile(s.bootstrapPath, []byte(password+"\n"), 0o600)
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
