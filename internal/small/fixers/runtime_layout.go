package fixers

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/justyn-clark/small-protocol/internal/small"
)

type RuntimeLayoutMigration struct {
	SourceRoot string
	TargetRoot string
	Moved      []string
}

type RuntimeLayoutFixResult struct {
	Migrations []RuntimeLayoutMigration
	Deduped    []string
}

func FixRuntimeLayout(baseDir string) (RuntimeLayoutFixResult, error) {
	plans := []struct {
		source string
		target string
	}{
		{source: small.LegacyArchiveDir(baseDir), target: small.ArchiveStoreDir(baseDir)},
		{source: small.LegacyRunsDir(baseDir), target: small.RunStoreDir(baseDir)},
	}

	result := RuntimeLayoutFixResult{}
	for _, plan := range plans {
		migration, deduped, err := migrateLegacyTree(baseDir, plan.source, plan.target)
		if err != nil {
			return RuntimeLayoutFixResult{}, err
		}
		if len(migration.Moved) > 0 {
			result.Migrations = append(result.Migrations, migration)
		}
		result.Deduped = append(result.Deduped, deduped...)
	}

	return result, nil
}

func migrateLegacyTree(baseDir, sourceRoot, targetRoot string) (RuntimeLayoutMigration, []string, error) {
	info, err := os.Stat(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return RuntimeLayoutMigration{}, nil, nil
		}
		return RuntimeLayoutMigration{}, nil, err
	}
	if !info.IsDir() {
		return RuntimeLayoutMigration{}, nil, fmt.Errorf("legacy runtime path is not a directory: %s", sourceRoot)
	}

	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return RuntimeLayoutMigration{}, nil, err
	}

	migration := RuntimeLayoutMigration{
		SourceRoot: relPath(baseDir, sourceRoot),
		TargetRoot: relPath(baseDir, targetRoot),
	}
	var deduped []string
	if err := mergeLegacyEntries(baseDir, sourceRoot, targetRoot, &migration, &deduped); err != nil {
		return RuntimeLayoutMigration{}, nil, err
	}
	if err := removeIfEmpty(sourceRoot); err != nil {
		return RuntimeLayoutMigration{}, nil, err
	}
	return migration, deduped, nil
}

func mergeLegacyEntries(baseDir, sourceDir, targetDir string, migration *RuntimeLayoutMigration, deduped *[]string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		targetPath := filepath.Join(targetDir, entry.Name())

		if entry.IsDir() {
			if err := moveLegacyDir(baseDir, sourcePath, targetPath, migration, deduped); err != nil {
				return err
			}
			continue
		}

		if err := moveLegacyFile(baseDir, sourcePath, targetPath, migration, deduped); err != nil {
			return err
		}
	}

	return nil
}

func moveLegacyDir(baseDir, sourcePath, targetPath string, migration *RuntimeLayoutMigration, deduped *[]string) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(sourcePath, targetPath); err != nil {
			return err
		}
		migration.Moved = append(migration.Moved, fmt.Sprintf("%s -> %s", relPath(baseDir, sourcePath), relPath(baseDir, targetPath)))
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot migrate %s to %s: destination exists and is not a directory", relPath(baseDir, sourcePath), relPath(baseDir, targetPath))
	}
	if err := mergeLegacyEntries(baseDir, sourcePath, targetPath, migration, deduped); err != nil {
		return err
	}
	return removeIfEmpty(sourcePath)
}

func moveLegacyFile(baseDir, sourcePath, targetPath string, migration *RuntimeLayoutMigration, deduped *[]string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(targetPath); err == nil {
		if !bytes.Equal(data, existing) {
			return fmt.Errorf("cannot migrate %s to %s: destination already exists with different content", relPath(baseDir, sourcePath), relPath(baseDir, targetPath))
		}
		if err := os.Remove(sourcePath); err != nil {
			return err
		}
		*deduped = append(*deduped, relPath(baseDir, sourcePath))
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return err
	}
	migration.Moved = append(migration.Moved, fmt.Sprintf("%s -> %s", relPath(baseDir, sourcePath), relPath(baseDir, targetPath)))
	return nil
}

func removeIfEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) != 0 {
		return nil
	}
	return os.Remove(path)
}

func relPath(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
