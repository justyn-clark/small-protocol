package small

import "path/filepath"

const (
	CacheDirName = ".small-cache"
)

func CacheDir(baseDir string) string {
	return filepath.Join(baseDir, CacheDirName)
}

func CacheDraftsDir(baseDir string) string {
	return filepath.Join(CacheDir(baseDir), "drafts")
}

func CacheCommandLogsDir(baseDir, replayID string) string {
	return filepath.Join(CacheDir(baseDir), "logs", replayID, "commands")
}
