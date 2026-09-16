package sessionv2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
)

const MaxPortableEvidenceBytes = 64 << 20

type EvidenceVerification struct {
	ReceiptDigest string `json:"receipt_digest"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
	ArtifactPath  string `json:"artifact_path,omitempty"`
}

func SavePortableEvidence(baseDir, sourcePath string, receipt Receipt) (Receipt, error) {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return Receipt{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Receipt{}, fmt.Errorf("evidence source must be a regular non-symlink file")
	}
	if info.Size() > MaxPortableEvidenceBytes {
		return Receipt{}, fmt.Errorf("evidence exceeds %d-byte limit", MaxPortableEvidenceBytes)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return Receipt{}, err
	}
	sum := sha256.Sum256(data)
	artifactDigest := hex.EncodeToString(sum[:])
	artifactRel := filepath.Join(".small", "evidence", "blobs", "sha256", artifactDigest)
	if err := ensureEvidenceParentsSafe(baseDir, artifactRel); err != nil {
		return Receipt{}, err
	}
	var prepared Receipt
	err = small.WithStateLock(baseDir, func() error {
		store, err := Load(baseDir)
		if err != nil {
			return err
		}
		receipt.SmallVersion = ProfileVersion
		receipt.ProjectID = store.Profile.ProjectID
		if receipt.ReceiptID == "" {
			receipt.ReceiptID, err = NewID("receipt")
			if err != nil {
				return err
			}
		}
		receipt.Strength = "verified_artifact"
		receipt.Availability = "verified"
		receipt.ArtifactPath = filepath.ToSlash(artifactRel)
		receipt.ArtifactDigest = artifactDigest
		receipt.ArtifactSize = int64(len(data))
		receipt.Digest, err = DigestRecord(receipt)
		if err != nil {
			return err
		}
		if err := validateReceipt(store.Profile, receipt); err != nil {
			return err
		}
		_, err = small.WriteStateFilesLocked(baseDir, []small.StateFile{{Path: artifactRel, Data: data}, {Path: filepath.Join(".small", "receipts", "sha256", receipt.Digest+".json"), Data: append(mustCanonical(receipt), '\n')}})
		if err == nil {
			prepared = receipt
		}
		return err
	})
	return prepared, err
}

func VerifyEvidence(baseDir string) ([]EvidenceVerification, error) {
	store, err := Load(baseDir)
	if err != nil {
		return nil, err
	}
	results := make([]EvidenceVerification, 0, len(store.Receipts))
	for digest, receipt := range store.Receipts {
		result := EvidenceVerification{ReceiptDigest: digest, ArtifactPath: receipt.ArtifactPath}
		if receipt.ArtifactPath == "" {
			result.Status = receipt.Availability
			results = append(results, result)
			continue
		}
		path, err := safeEvidencePath(baseDir, receipt.ArtifactPath)
		if err != nil {
			result.Status = "failed"
			result.Reason = err.Error()
			results = append(results, result)
			continue
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			result.Status = "unavailable"
			result.Reason = "portable artifact is missing"
			results = append(results, result)
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			result.Status = "failed"
			result.Reason = "artifact is not a regular non-symlink file"
			results = append(results, result)
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			result.Status = "failed"
			result.Reason = err.Error()
			results = append(results, result)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != receipt.ArtifactDigest || int64(len(data)) != receipt.ArtifactSize {
			result.Status = "failed"
			result.Reason = "artifact digest or size mismatch"
		} else {
			result.Status = "verified"
		}
		results = append(results, result)
	}
	sortEvidenceResults(results)
	return results, nil
}

func safeEvidencePath(baseDir, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel)))
	required := filepath.Join(".small", "evidence") + string(filepath.Separator)
	if filepath.IsAbs(clean) || !strings.HasPrefix(clean, required) || clean == filepath.Join(".small", "evidence") {
		return "", fmt.Errorf("unsafe evidence artifact path %q", rel)
	}
	path := filepath.Join(baseDir, clean)
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		root, rootErr := filepath.EvalSymlinks(filepath.Join(baseDir, ".small", "evidence"))
		if rootErr != nil {
			return "", rootErr
		}
		root += string(filepath.Separator)
		if !strings.HasPrefix(resolved, root) {
			return "", fmt.Errorf("evidence path escapes through symlink")
		}
	}
	return path, nil
}

func ensureEvidenceParentsSafe(baseDir, rel string) error {
	current := baseDir
	parts := strings.Split(filepath.Clean(rel), string(filepath.Separator))
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("evidence destination parent is a symlink: %s", current)
		}
	}
	return nil
}
func sortEvidenceResults(results []EvidenceVerification) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].ReceiptDigest < results[i].ReceiptDigest {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
