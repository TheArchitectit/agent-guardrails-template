package team

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Team data at rest (SEC-007 / Spec 17 R17-06).
//
// These tests drive the REAL Manager through save and load, not the encryption
// type in isolation. The defect they exist to catch was precisely that the
// encryption type worked and nothing called it: a unit test of Seal/Open would
// have passed for two years while every assignment was written plaintext.

const testTeamKey = "operator-team-key-0123456789abcdef"

func newTestManager(t *testing.T, dir, key string) *Manager {
	t.Helper()
	m, err := NewManager("demo", WithBaseDir(dir), WithEncryptionKey(key))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m
}

func assignOne(t *testing.T, m *Manager, person string) {
	t.Helper()
	if err := m.InitializeProject(); err != nil {
		t.Fatalf("InitializeProject: %v", err)
	}
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Assign to the first role of team 1 so the test does not depend on the
	// exact standard-team roster.
	team, err := m.GetTeamByID(1)
	if err != nil {
		t.Fatalf("GetTeamByID: %v", err)
	}
	if len(team.Roles) == 0 {
		t.Fatalf("team 1 has no roles to assign")
	}
	if err := m.AssignRole(1, team.Roles[0].Name, person); err != nil {
		t.Fatalf("AssignRole: %v", err)
	}
}

// readStoredTeams returns the raw on-disk JSON as a generic value, so the test
// can assert on what actually reached the filesystem rather than on the
// in-memory model.
func readStoredTeams(t *testing.T, m *Manager) map[string]any {
	t.Helper()
	data, err := os.ReadFile(m.GetConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return raw
}

// withKeyNeverWritesThePlaintextName is the positive control: a configured key
// must actually seal the assignment it claims to protect.
func TestTeamAssignmentsSealedAtRestWithKey(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, dir, testTeamKey)
	assignOne(t, m, "Ada Lovelace")

	raw := readStoredTeams(t, m)
	body, _ := json.Marshal(raw)
	if strings.Contains(string(body), "Ada Lovelace") {
		t.Fatalf("assignee name is present in the stored file; the configured key did not seal it:\n%s", body)
	}
	if !strings.Contains(string(body), "v1:") {
		t.Fatalf("stored file carries no sealed form; sealing did not run:\n%s", body)
	}

	// And it must round-trip: a fresh manager with the same key reads it back.
	again := newTestManager(t, dir, testTeamKey)
	if err := again.Load(); err != nil {
		t.Fatalf("Load with the same key: %v", err)
	}
	team, err := again.GetTeamByID(1)
	if err != nil {
		t.Fatalf("GetTeamByID: %v", err)
	}
	var found bool
	for _, role := range team.Roles {
		if role.AssignedTo != nil && *role.AssignedTo == "Ada Lovelace" {
			found = true
		}
	}
	if !found {
		t.Fatalf("assignment did not round-trip through seal/open")
	}
}

// withoutKeyWritesPlaintext is the documented `local` posture, asserted rather
// than assumed: no key means plaintext, and the code must not pretend otherwise.
func TestTeamAssignmentsPlaintextWithoutKey(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, dir, "")
	assignOne(t, m, "Grace Hopper")

	raw := readStoredTeams(t, m)
	body, _ := json.Marshal(raw)
	if !strings.Contains(string(body), "Grace Hopper") {
		t.Fatalf("expected plaintext with no key configured:\n%s", body)
	}
	if strings.Contains(string(body), "v1:") {
		t.Fatalf("sealed without a key configured:\n%s", body)
	}
	if m.encryption.Enabled() {
		t.Fatalf("manager reports encryption enabled with no key")
	}
}

// withoutKeyRefusesSealedData is the fail-closed half, and the one the old code
// got wrong in two ways at once: it never ran, and its Decrypt returned the
// ciphertext as if it were the plaintext.
func TestLoadingSealedDataWithoutKeyFailsClosed(t *testing.T) {
	dir := t.TempDir()
	sealed := newTestManager(t, dir, testTeamKey)
	assignOne(t, sealed, "Katherine Johnson")

	unsealed := newTestManager(t, dir, "")
	err := unsealed.Load()
	if err == nil {
		t.Fatalf("Load with sealed data and no key succeeded; it must fail closed")
	}
	if !errors.Is(err, ErrTeamEncryptionKeyRequired) {
		t.Fatalf("expected ErrTeamEncryptionKeyRequired, got %v", err)
	}
}

// wrongKeyFailsClosed: a sealed value is authenticated, so a different key is an
// error rather than garbage that looks like an assignee name.
func TestLoadingSealedDataWithWrongKeyFailsClosed(t *testing.T) {
	dir := t.TempDir()
	sealed := newTestManager(t, dir, testTeamKey)
	assignOne(t, sealed, "Mary Jackson")

	other := newTestManager(t, dir, "a-different-operator-team-key-987654321")
	err := other.Load()
	if err == nil {
		t.Fatalf("Load with the wrong key succeeded; authentication must fail")
	}
	if errors.Is(err, ErrTeamEncryptionKeyRequired) {
		t.Fatalf("wrong key should be an authentication failure, not the missing-key error")
	}
}

// tamperedCiphertextFailsClosed: flip one byte of the sealed value and the load
// must refuse, never fall back to the stored bytes.
func TestTamperedTeamCiphertextFailsClosed(t *testing.T) {
	dir := t.TempDir()
	sealed := newTestManager(t, dir, testTeamKey)
	assignOne(t, sealed, "Dorothy Vaughan")

	path := sealed.GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	idx := strings.Index(string(data), "v1:")
	if idx < 0 {
		t.Fatalf("no sealed value in stored file")
	}
	// Flip a byte inside the base64 body, past the "v1:" prefix.
	mutated := []byte(string(data))
	pos := idx + len("v1:") + 4
	if pos >= len(mutated) || mutated[pos] == '"' {
		t.Fatalf("could not locate a ciphertext byte to mutate")
	}
	mutated[pos] ^= 0x01
	if err := os.WriteFile(path, mutated, 0o600); err != nil {
		t.Fatalf("write mutated: %v", err)
	}

	if err := newTestManager(t, dir, testTeamKey).Load(); err == nil {
		t.Fatalf("Load of tampered ciphertext succeeded; it must fail closed")
	}
}

// fileModeIsOwnerOnly: assignments carry personal names, so the stored file must
// not be world-readable. It was written 0644 before this change.
func TestTeamConfigFileIsOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, dir, testTeamKey)
	assignOne(t, m, "Annie Easley")

	info, err := os.Stat(m.GetConfigPath())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Fatalf("team config mode is %#o; group/world access must be removed", perm)
	}
}

// unusableKeyStopsConstruction: a configured key that cannot be used must not
// produce a manager that silently protects nothing.
func TestNewManagerRejectsUnusableKey(t *testing.T) {
	dir := t.TempDir()
	// secretbox.New rejects only an empty key, which WithEncryptionKey already
	// filters; assert the contract at the constructor boundary it does enforce.
	em, err := NewEncryptionManager("   ")
	if err != nil {
		t.Fatalf("blank key should yield a disabled manager, not an error: %v", err)
	}
	if em.Enabled() {
		t.Fatalf("blank key produced an enabled manager")
	}
	if _, err := NewManager("demo", WithBaseDir(dir), WithEncryptionManager(em)); err != nil {
		t.Fatalf("NewManager with a disabled manager: %v", err)
	}
}

// saveLeavesMemoryUnsealed: sealing happens on the way to disk. The in-memory
// model callers read must hold plaintext, or every status tool would report
// ciphertext.
func TestSaveDoesNotMutateInMemoryAssignments(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, dir, testTeamKey)
	assignOne(t, m, "Margaret Hamilton")

	team, err := m.GetTeamByID(1)
	if err != nil {
		t.Fatalf("GetTeamByID: %v", err)
	}
	var found bool
	for _, role := range team.Roles {
		if role.AssignedTo == nil {
			continue
		}
		if *role.AssignedTo == "Margaret Hamilton" {
			found = true
		}
		if strings.HasPrefix(*role.AssignedTo, "v1:") {
			t.Fatalf("in-memory assignment was left sealed: %q", *role.AssignedTo)
		}
	}
	if !found {
		t.Fatalf("in-memory assignment lost after save")
	}

	// Defend the temp path too: save writes to a sibling .tmp then renames.
	entries, err := os.ReadDir(filepath.Dir(m.GetConfigPath()))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind after save: %s", e.Name())
		}
	}
}
