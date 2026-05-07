package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"smart2fa/internal/auth"
	"smart2fa/internal/crypto"
	"smart2fa/internal/totp"
	"smart2fa/internal/vault"
)

// extractGroups returns sorted unique group names, "Default" always first.
func extractGroups(entries []vault.Entry) ([]string, map[string]int) {
	counts := map[string]int{}
	for _, e := range entries {
		g := e.Group
		if g == "" {
			g = "Default"
		}
		counts[g]++
	}
	counts["Default"] += 0 // ensure Default always exists
	groups := []string{"Default"}
	extra := []string{}
	for g := range counts {
		if g != "Default" {
			extra = append(extra, g)
		}
	}
	sort.Strings(extra)
	groups = append(groups, extra...)
	return groups, counts
}

// requireVault is a helper that gets the session token + runtime, or redirects.
// For HTMX requests it uses the HX-Redirect header so the full page navigates
// instead of swapping the lock HTML into a partial container.
func requireVault(w http.ResponseWriter, r *http.Request) (string, *vault.VaultRuntime, bool) {
	redirect := func() {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}

	token, ok := auth.GetToken(r)
	if !ok {
		redirect()
		return "", nil, false
	}
	rt, ok := vault.GetRuntime(token)
	if !ok {
		redirect()
		return "", nil, false
	}
	return token, rt, true
}

// GetLock serves the lock screen.
func GetLock(tmpl TmplRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Already unlocked?
		if token, ok := auth.GetToken(r); ok {
			if _, ok := vault.GetRuntime(token); ok {
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
				return
			}
		}
		tmpl.Render(w, "lock.html", nil)
	}
}

// PostUnlock handles phrase + passcode submission.
func PostUnlock(tmpl TmplRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := strings.TrimSpace(r.FormValue("phrase"))
		passcode := strings.TrimSpace(r.FormValue("passcode"))

		if phrase == "" || passcode == "" {
			tmpl.Render(w, "lock.html", map[string]any{"Error": "Phrase and passcode are required."})
			return
		}

		token, err := auth.NewToken()
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}

		if _, err := vault.Unlock(phrase, passcode, token); err != nil {
			tmpl.Render(w, "lock.html", map[string]any{"Error": "Failed to unlock vault."})
			return
		}

		auth.SetCookie(w, token)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// GetDashboard renders the dashboard.
func GetDashboard(tmpl TmplRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}
		groups, counts := extractGroups(rt.Entries)
		tmpl.Render(w, "dashboard.html", map[string]any{
			"Groups":      groups,
			"GroupCounts": counts,
		})
	}
}

// GetCodes returns the HTMX partial for live TOTP codes.
func GetCodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}

		type codeRow struct {
			Name      string
			Code      string
			Group     string
			Remaining int
		}

		// Filter by group
		group := r.URL.Query().Get("group")
		if group == "" {
			group = "Default"
		}

		remaining := totp.TimeRemaining()
		rows := make([]codeRow, 0)
		for _, e := range rt.Entries {
			g := e.Group
			if g == "" {
				g = "Default"
			}
			if g != group {
				continue
			}
			rawCode, err := totp.GenerateCode(e.Secret)
			if err != nil {
				log.Printf("[TOTP] entry=%q secret=%q err=%v", e.Name, e.Secret, err)
			}
			// Format as "XXX XXX" for readability
			displayCode := rawCode
			if len(rawCode) == 6 {
				displayCode = rawCode[:3] + " " + rawCode[3:]
			}
			rows = append(rows, codeRow{Name: e.Name, Code: displayCode, Group: g, Remaining: remaining})
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if len(rows) == 0 {
			fmt.Fprint(w, `<div class="empty-state"><div class="empty-icon">🔐</div><p>No accounts yet.</p><p class="empty-sub">Click <strong>Add Account</strong> to get started.</p></div>`)
			return
		}
		// barPct = percentage of 30s window remaining (shrinks from 100→0)
		barPct := float64(remaining) / 30.0 * 100.0
		for _, row := range rows {
			fmt.Fprintf(w, `<div class="code-row" data-name="%s">
  <span class="entry-name">%s</span>
  <span class="entry-code">%s</span>
  <div class="timer-bar-wrap"><div class="timer-bar" style="width:%.1f%%" data-remaining="%d"></div></div>
  <button class="btn-copy" onclick="copyCode(this,'%s')">Copy</button>
  <button class="btn-edit" onclick="openEditModal(this)" data-name="%s" data-group="%s" title="Edit">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
  </button>
  <form action="/entry/delete" method="POST" class="delete-form" onsubmit="return confirm('Remove %s?')">
    <input type="hidden" name="name" value="%s"/>
    <button type="submit" class="btn-danger">✕</button>
  </form>
</div>`, row.Name, row.Name, row.Code, barPct, remaining, row.Code, row.Name, row.Group, row.Name, row.Name)
		}
	}
}

// PostAddEntry adds a new TOTP entry.
func PostAddEntry(tmpl TmplRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		secret := strings.TrimSpace(r.FormValue("secret"))

		// Handle otpauth:// URI
		if strings.HasPrefix(secret, "otpauth://") {
			parsedName, parsedSecret, err := totp.ParseOTPAuth(secret)
			if err == nil {
				if name == "" {
					name = parsedName
				}
				secret = parsedSecret
			}
		}

		if name == "" || secret == "" {
			http.Redirect(w, r, "/dashboard?error=invalid", http.StatusSeeOther)
			return
		}

		// Normalize secret: strip whitespace + uppercase
		secret = strings.ToUpper(strings.Map(func(r rune) rune {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
				return -1
			}
			return r
		}, secret))

		// Read group, default to "Default"
		group := strings.TrimSpace(r.FormValue("group"))
		if group == "" {
			group = "Default"
		}

		rt.Entries = append(rt.Entries, vault.Entry{Name: name, Secret: secret, Group: group})
		if err := vault.Save(rt); err != nil {
			http.Error(w, "save failed", 500)
			return
		}
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// PostEditEntry updates an existing TOTP entry.
func PostEditEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}

		origName := strings.TrimSpace(r.FormValue("orig_name"))
		newName  := strings.TrimSpace(r.FormValue("name"))
		newSecret := strings.TrimSpace(r.FormValue("secret"))
		newGroup := strings.TrimSpace(r.FormValue("group"))

		if origName == "" || newName == "" {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		if newGroup == "" {
			newGroup = "Default"
		}

		// Normalize new secret if provided
		if newSecret != "" {
			newSecret = strings.ToUpper(strings.Map(func(r rune) rune {
				if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
					return -1
				}
				return r
			}, newSecret))
		}

		for i, e := range rt.Entries {
			if e.Name == origName {
				rt.Entries[i].Name = newName
				rt.Entries[i].Group = newGroup
				if newSecret != "" {
					rt.Entries[i].Secret = newSecret
				}
				break
			}
		}

		if err := vault.Save(rt); err != nil {
			http.Error(w, "save failed", 500)
			return
		}
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// PostDeleteEntry removes a TOTP entry by name.
func PostDeleteEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		newEntries := rt.Entries[:0]
		for _, e := range rt.Entries {
			if e.Name != name {
				newEntries = append(newEntries, e)
			}
		}
		rt.Entries = newEntries
		vault.Save(rt)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// PostRenameGroup renames all entries in a group from old_name → new_name.
func PostRenameGroup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}

		oldName := strings.TrimSpace(r.FormValue("old_name"))
		newName := strings.TrimSpace(r.FormValue("new_name"))

		if oldName == "" || newName == "" || oldName == newName {
			w.WriteHeader(http.StatusOK)
			return
		}
		if oldName == "Default" {
			http.Error(w, "Cannot rename Default group", http.StatusBadRequest)
			return
		}

		for i, e := range rt.Entries {
			g := e.Group
			if g == "" {
				g = "Default"
			}
			if g == oldName {
				rt.Entries[i].Group = newName
			}
		}

		if err := vault.Save(rt); err != nil {
			http.Error(w, "save failed", 500)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// PostLock clears session and redirects to lock screen.
func PostLock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token, ok := auth.GetToken(r); ok {
			vault.DeleteRuntime(token)
		}
		auth.ClearCookie(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// --- Backup ---

// GetBackupPage serves the export/import backup page.
func GetBackupPage(tmpl TmplRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, ok := requireVault(w, r)
		if !ok {
			return
		}
		tmpl.Render(w, "backup.html", nil)
	}
}

// PostExport downloads an encrypted .votp backup file.
func PostExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}

		backupPass := strings.TrimSpace(r.FormValue("backup_password"))
		if backupPass == "" {
			http.Error(w, "backup password required", 400)
			return
		}

		type backupContent struct {
			Entries []vault.Entry `json:"entries"`
		}
		plain, _ := json.Marshal(backupContent{Entries: rt.Entries})

		salt, _ := crypto.NewSalt()
		key := crypto.DeriveKey(backupPass, "", salt)
		ciphertext, err := crypto.Encrypt(key, plain)
		if err != nil {
			http.Error(w, "encrypt error", 500)
			return
		}

		type backupFile struct {
			Version    int    `json:"version"`
			KDFType    string `json:"kdf_type"`
			Salt       []byte `json:"salt"`
			Ciphertext []byte `json:"ciphertext"`
		}
		bf := backupFile{Version: 1, KDFType: "argon2id", Salt: salt, Ciphertext: ciphertext}
		out, _ := json.Marshal(bf)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\"backup.votp\"")
		w.Write(out)
	}
}

// PostImport imports a .votp backup file.
func PostImport(tmpl TmplRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, rt, ok := requireVault(w, r)
		if !ok {
			return
		}

		backupPass := strings.TrimSpace(r.FormValue("backup_password"))
		file, _, err := r.FormFile("votp_file")
		if err != nil {
			tmpl.Render(w, "backup.html", map[string]any{"Error": "No file uploaded."})
			return
		}
		defer file.Close()

		var bf struct {
			Version    int    `json:"version"`
			KDFType    string `json:"kdf_type"`
			Salt       []byte `json:"salt"`
			Ciphertext []byte `json:"ciphertext"`
		}
		if err := json.NewDecoder(file).Decode(&bf); err != nil {
			tmpl.Render(w, "backup.html", map[string]any{"Error": "Invalid backup file."})
			return
		}

		key := crypto.DeriveKey(backupPass, "", bf.Salt)
		plain, err := crypto.Decrypt(key, bf.Ciphertext)
		if err != nil {
			tmpl.Render(w, "backup.html", map[string]any{"Error": "Wrong backup password or corrupted file."})
			return
		}

		var content struct {
			Entries []vault.Entry `json:"entries"`
		}
		if err := json.Unmarshal(plain, &content); err != nil {
			tmpl.Render(w, "backup.html", map[string]any{"Error": "Corrupted backup content."})
			return
		}

		// Merge entries (no duplicates by name)
		existing := map[string]bool{}
		for _, e := range rt.Entries {
			existing[e.Name] = true
		}
		for _, e := range content.Entries {
			if !existing[e.Name] {
				rt.Entries = append(rt.Entries, e)
			}
		}
		vault.Save(rt)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// TmplRenderer interface so handlers can render templates.
type TmplRenderer interface {
	Render(w http.ResponseWriter, name string, data any)
}
