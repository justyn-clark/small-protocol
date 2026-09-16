package small

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const stateTransactionVersion = 1

var ErrStateBusy = errors.New("SMALL state is busy")

// StateFile is one repository-relative file in a recoverable state transition.
// State transactions are deliberately restricted to .small/.
type StateFile struct {
	Path string
	Data []byte
	Mode os.FileMode
}

type transactionManifest struct {
	Version   int                   `json:"version"`
	ID        string                `json:"id"`
	Phase     string                `json:"phase"`
	CreatedAt string                `json:"created_at"`
	Files     []transactionFileMeta `json:"files"`
}

type transactionFileMeta struct {
	Path      string `json:"path"`
	OldExists bool   `json:"old_exists"`
	OldDigest string `json:"old_digest,omitempty"`
	NewDigest string `json:"new_digest"`
	Mode      uint32 `json:"mode"`
}

// WithStateLock serializes cooperative writers in one checkout. The lock is
// local only: it is not a lease and makes no claim about disconnected clones.
func WithStateLock(baseDir string, fn func() error) error {
	cacheDir := filepath.Join(baseDir, ".small-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("create SMALL cache: %w", err)
	}
	lockPath := filepath.Join(cacheDir, "state.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: lock %s already exists; another local writer may be active", ErrStateBusy, lockPath)
		}
		return fmt.Errorf("create SMALL state lock: %w", err)
	}
	_, _ = fmt.Fprintf(lock, "pid=%d\ncreated_at=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
	if err := lock.Sync(); err != nil {
		_ = lock.Close()
		_ = os.Remove(lockPath)
		return fmt.Errorf("sync SMALL state lock: %w", err)
	}
	_ = lock.Close()
	defer os.Remove(lockPath)

	if err := RecoverStateTransactionsLocked(baseDir); err != nil {
		return err
	}
	return fn()
}

// WriteStateFiles publishes one or more .small files under the local writer
// lock. Equivalent bytes are a no-op and do not touch timestamps.
func WriteStateFiles(baseDir string, files []StateFile) (bool, error) {
	changed := false
	err := WithStateLock(baseDir, func() error {
		var err error
		changed, err = WriteStateFilesLocked(baseDir, files)
		return err
	})
	return changed, err
}

// WriteStateFilesLocked requires the caller to hold WithStateLock.
func WriteStateFilesLocked(baseDir string, files []StateFile) (bool, error) {
	if len(files) == 0 {
		return false, nil
	}
	normalized := make([]StateFile, 0, len(files))
	seen := map[string]struct{}{}
	for _, file := range files {
		rel, err := validStateRelativePath(file.Path)
		if err != nil {
			return false, err
		}
		if _, ok := seen[rel]; ok {
			return false, fmt.Errorf("duplicate state transaction path %s", rel)
		}
		seen[rel] = struct{}{}
		file.Path = rel
		if file.Mode == 0 {
			file.Mode = 0o644
		}
		current, err := os.ReadFile(filepath.Join(baseDir, rel))
		if err == nil && string(current) == string(file.Data) {
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("read current %s: %w", rel, err)
		}
		normalized = append(normalized, file)
	}
	if len(normalized) == 0 {
		return false, nil
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Path < normalized[j].Path })

	id, err := randomStateID()
	if err != nil {
		return false, err
	}
	txnDir := filepath.Join(baseDir, ".small-cache", "transactions", id)
	if err := os.MkdirAll(filepath.Join(txnDir, "old"), 0o700); err != nil {
		return false, fmt.Errorf("create state transaction: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(txnDir, "new"), 0o700); err != nil {
		return false, fmt.Errorf("create state transaction: %w", err)
	}

	manifest := transactionManifest{Version: stateTransactionVersion, ID: id, Phase: "prepared", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	for idx, file := range normalized {
		target := filepath.Join(baseDir, file.Path)
		meta := transactionFileMeta{Path: file.Path, NewDigest: digestBytes(file.Data), Mode: uint32(file.Mode.Perm())}
		if current, readErr := os.ReadFile(target); readErr == nil {
			meta.OldExists = true
			meta.OldDigest = digestBytes(current)
			if err := writeDurableFile(filepath.Join(txnDir, "old", fmt.Sprintf("%06d", idx)), current, file.Mode); err != nil {
				return false, err
			}
		} else if !os.IsNotExist(readErr) {
			return false, fmt.Errorf("snapshot %s: %w", file.Path, readErr)
		}
		if err := writeDurableFile(filepath.Join(txnDir, "new", fmt.Sprintf("%06d", idx)), file.Data, file.Mode); err != nil {
			return false, err
		}
		manifest.Files = append(manifest.Files, meta)
	}
	if err := writeManifest(txnDir, manifest); err != nil {
		return false, err
	}
	if shouldFailTransaction("prepared") {
		return false, fmt.Errorf("injected state transaction failure after prepared; recoverable transaction %s retained", id)
	}

	for idx, meta := range manifest.Files {
		target := filepath.Join(baseDir, meta.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return false, fmt.Errorf("create target directory: %w", err)
		}
		newPath := filepath.Join(txnDir, "new", fmt.Sprintf("%06d", idx))
		if err := os.Rename(newPath, target); err != nil {
			return false, fmt.Errorf("publish %s: %w; recoverable transaction %s retained", meta.Path, err, id)
		}
		if err := syncDirectory(filepath.Dir(target)); err != nil {
			return false, fmt.Errorf("sync published %s: %w; recoverable transaction %s retained", meta.Path, err, id)
		}
		if shouldFailTransaction(fmt.Sprintf("publish-%d", idx+1)) {
			return false, fmt.Errorf("injected state transaction failure after publication %d; recoverable transaction %s retained", idx+1, id)
		}
	}
	manifest.Phase = "committed"
	if err := writeManifest(txnDir, manifest); err != nil {
		return false, fmt.Errorf("mark transaction committed: %w; recoverable transaction %s retained", err, id)
	}
	if shouldFailTransaction("committed") {
		return false, fmt.Errorf("injected state transaction failure after committed; recoverable transaction %s retained", id)
	}
	if err := os.RemoveAll(txnDir); err != nil {
		return true, fmt.Errorf("state committed but transaction cleanup failed: %w", err)
	}
	return true, nil
}

// PendingStateTransactions returns recoverable journal IDs without mutating.
func PendingStateTransactions(baseDir string) ([]string, error) {
	root := filepath.Join(baseDir, ".small-cache", "transactions")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// RecoverStateTransactions completes any prepared journal. It is safe to call
// repeatedly; authoritative target bytes are verified before cleanup.
func RecoverStateTransactions(baseDir string) error {
	return WithStateLock(baseDir, func() error { return nil })
}

func RecoverStateTransactionsLocked(baseDir string) error {
	ids, err := PendingStateTransactions(baseDir)
	if err != nil {
		return fmt.Errorf("list state transactions: %w", err)
	}
	for _, id := range ids {
		txnDir := filepath.Join(baseDir, ".small-cache", "transactions", id)
		manifest, err := readManifest(txnDir)
		if err != nil {
			return fmt.Errorf("recover transaction %s: %w", id, err)
		}
		for idx, meta := range manifest.Files {
			target := filepath.Join(baseDir, meta.Path)
			if current, readErr := os.ReadFile(target); readErr == nil && digestBytes(current) == meta.NewDigest {
				continue
			}
			newPath := filepath.Join(txnDir, "new", fmt.Sprintf("%06d", idx))
			data, readErr := os.ReadFile(newPath)
			if readErr != nil {
				return fmt.Errorf("recover transaction %s target %s: new bytes unavailable", id, meta.Path)
			}
			if digestBytes(data) != meta.NewDigest {
				return fmt.Errorf("recover transaction %s target %s: staged digest mismatch", id, meta.Path)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Rename(newPath, target); err != nil {
				return fmt.Errorf("recover transaction %s target %s: %w", id, meta.Path, err)
			}
			if err := syncDirectory(filepath.Dir(target)); err != nil {
				return err
			}
		}
		if err := os.RemoveAll(txnDir); err != nil {
			return fmt.Errorf("cleanup recovered transaction %s: %w", id, err)
		}
	}
	return nil
}

func validStateRelativePath(value string) (string, error) {
	rel := filepath.Clean(strings.TrimSpace(value))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid state transaction path %q", value)
	}
	prefix := SmallDir + string(filepath.Separator)
	if !strings.HasPrefix(rel, prefix) {
		return "", fmt.Errorf("state transaction path must be under %s: %s", SmallDir, rel)
	}
	return rel, nil
}

func writeManifest(txnDir string, manifest transactionManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := filepath.Join(txnDir, "manifest.json.tmp")
	if err := writeDurableFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(txnDir, "manifest.json")); err != nil {
		return err
	}
	return syncDirectory(txnDir)
}

func readManifest(txnDir string) (transactionManifest, error) {
	data, err := os.ReadFile(filepath.Join(txnDir, "manifest.json"))
	if err != nil {
		return transactionManifest{}, err
	}
	var manifest transactionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return transactionManifest{}, err
	}
	if manifest.Version != stateTransactionVersion || manifest.ID == "" || len(manifest.Files) == 0 {
		return transactionManifest{}, fmt.Errorf("invalid transaction manifest")
	}
	return manifest, nil
}

func writeDurableFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func randomStateID() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generate state transaction id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func shouldFailTransaction(boundary string) bool {
	return strings.TrimSpace(os.Getenv("SMALL_TEST_FAIL_TRANSACTION_AFTER")) == boundary
}
