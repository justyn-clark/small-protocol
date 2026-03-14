package small

import "path/filepath"

const (
	CacheDirName        = ".small-cache"
	RunStoreDirName     = ".small-runs"
	ArchiveStoreDirName = ".small-archive"
	RunIndexFileName    = "index.small.yml"
)

func CacheDir(baseDir string) string {
	return filepath.Join(baseDir, CacheDirName)
}

func RunStoreDir(baseDir string) string {
	return filepath.Join(baseDir, RunStoreDirName)
}

func ArchiveStoreDir(baseDir string) string {
	return filepath.Join(baseDir, ArchiveStoreDirName)
}

func RunIndexPath(baseDir string) string {
	return filepath.Join(RunStoreDir(baseDir), RunIndexFileName)
}

func CacheDraftsDir(baseDir string) string {
	return filepath.Join(CacheDir(baseDir), "drafts")
}

func CacheCommandLogsDir(baseDir, replayID string) string {
	return filepath.Join(CacheDir(baseDir), "logs", replayID, "commands")
}

func LegacyArchiveDir(baseDir string) string {
	return filepath.Join(baseDir, SmallDir, "archive")
}

func LegacyRunsDir(baseDir string) string {
	return filepath.Join(baseDir, SmallDir, "runs")
}
