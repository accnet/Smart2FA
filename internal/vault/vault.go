package vault

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"smart2fa/internal/crypto"
	"smart2fa/internal/db"
)

// Entry represents a single TOTP entry stored in the vault blob.
type Entry struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
	Group  string `json:"group,omitempty"` // empty = "Default"
}

// VaultRuntime holds decrypted vault data cached in RAM.
type VaultRuntime struct {
	VaultID int64
	Entries []Entry
	Key     []byte // derived encryption key (kept to re-encrypt on write)
}

var (
	mu    sync.RWMutex
	cache = map[string]*VaultRuntime{} // key: session token -> runtime
)

// GetRuntime returns a cached VaultRuntime for the given session token.
func GetRuntime(token string) (*VaultRuntime, bool) {
	mu.RLock()
	defer mu.RUnlock()
	r, ok := cache[token]
	return r, ok
}

// SetRuntime stores a VaultRuntime in the RAM cache.
func SetRuntime(token string, r *VaultRuntime) {
	mu.Lock()
	defer mu.Unlock()
	cache[token] = r
}

// DeleteRuntime removes a VaultRuntime from cache.
func DeleteRuntime(token string) {
	mu.Lock()
	defer mu.Unlock()
	delete(cache, token)
}

// Unlock finds or creates a vault, decrypts the blob, caches and returns the runtime.
func Unlock(phrase, passcode, sessionToken string) (*VaultRuntime, error) {
	vaultHash := crypto.VaultHash(phrase, passcode)

	var vaultID int64
	var salt []byte
	err := db.DB.QueryRow(`SELECT id, salt FROM vaults WHERE vault_hash = ?`, vaultHash).
		Scan(&vaultID, &salt)

	if err == sql.ErrNoRows {
		// Create new vault
		newSalt, err := crypto.NewSalt()
		if err != nil {
			return nil, err
		}
		now := time.Now().Unix()
		res, err := db.DB.Exec(
			`INSERT INTO vaults(vault_hash, salt, entries_count, last_access_at, created_at) VALUES(?,?,0,?,?)`,
			vaultHash, newSalt, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("create vault: %w", err)
		}
		vaultID, _ = res.LastInsertId()

		// Empty blob
		emptyBlob, _ := json.Marshal([]Entry{})
		key := crypto.DeriveKey(phrase, passcode, newSalt)
		encBlob, err := crypto.Encrypt(key, emptyBlob)
		if err != nil {
			return nil, err
		}
		_, err = db.DB.Exec(`INSERT INTO vault_blobs(vault_id, encrypted_blob) VALUES(?,?)`, vaultID, encBlob)
		if err != nil {
			return nil, err
		}

		rt := &VaultRuntime{VaultID: vaultID, Entries: []Entry{}, Key: key}
		SetRuntime(sessionToken, rt)
		return rt, nil
	} else if err != nil {
		return nil, fmt.Errorf("lookup vault: %w", err)
	}

	// Existing vault
	var encBlob []byte
	if err := db.DB.QueryRow(`SELECT encrypted_blob FROM vault_blobs WHERE vault_id = ?`, vaultID).
		Scan(&encBlob); err != nil {
		return nil, fmt.Errorf("get blob: %w", err)
	}

	key := crypto.DeriveKey(phrase, passcode, salt)
	plaintext, err := crypto.Decrypt(key, encBlob)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(plaintext, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// Update last_access_at
	db.DB.Exec(`UPDATE vaults SET last_access_at = ? WHERE id = ?`, time.Now().Unix(), vaultID)

	rt := &VaultRuntime{VaultID: vaultID, Entries: entries, Key: key}
	SetRuntime(sessionToken, rt)
	return rt, nil
}

// Save encrypts and persists the runtime's entries back to DB.
func Save(rt *VaultRuntime) error {
	plaintext, err := json.Marshal(rt.Entries)
	if err != nil {
		return err
	}
	encBlob, err := crypto.Encrypt(rt.Key, plaintext)
	if err != nil {
		return err
	}
	_, err = db.DB.Exec(
		`UPDATE vault_blobs SET encrypted_blob = ? WHERE vault_id = ?`,
		encBlob, rt.VaultID,
	)
	if err != nil {
		return err
	}
	db.DB.Exec(
		`UPDATE vaults SET entries_count = ?, last_access_at = ? WHERE id = ?`,
		len(rt.Entries), time.Now().Unix(), rt.VaultID,
	)
	return nil
}
