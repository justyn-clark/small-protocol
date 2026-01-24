package updatecheck

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultUpdateURL = "https://api.github.com/repos/justyn-clark/small-protocol/releases/latest"
	DefaultTTL       = 24 * time.Hour
	DefaultTimeout   = 2 * time.Second
)

type Options struct {
	CurrentVersion string
	UpdateURL      string
	CachePath      string
	Now            time.Time
	TTL            time.Duration
	Timeout        time.Duration
}

type Notice struct {
	Latest  string
	Current string
}

type cacheData struct {
	CheckedAt     string `json:"checked_at"`
	LatestVersion string `json:"latest_version"`
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

func Check(ctx context.Context, opts Options) (*Notice, error) {
	current := strings.TrimSpace(opts.CurrentVersion)
	if current == "" || current == "dev" {
		return nil, nil
	}

	cachePath, err := resolveCachePath(opts.CachePath)
	if err != nil {
		return nil, nil
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	ttl := opts.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}

	if cached, ok := loadCache(cachePath); ok {
		if !cacheExpired(cached, now, ttl) {
			if isNewerVersion(cached.LatestVersion, current) {
				return &Notice{Latest: normalizeVersion(cached.LatestVersion), Current: normalizeVersion(current)}, nil
			}
			return nil, nil
		}
	}

	latest, err := fetchLatestRelease(ctx, opts)
	if err != nil || latest == "" {
		return nil, nil
	}

	_ = writeCache(cachePath, cacheData{CheckedAt: now.UTC().Format(time.RFC3339Nano), LatestVersion: latest})

	if isNewerVersion(latest, current) {
		return &Notice{Latest: normalizeVersion(latest), Current: normalizeVersion(current)}, nil
	}

	return nil, nil
}

func resolveCachePath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "small", "updatecheck.json"), nil
}

func loadCache(path string) (cacheData, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheData{}, false
	}
	var cached cacheData
	if err := json.Unmarshal(data, &cached); err != nil {
		return cacheData{}, false
	}
	return cached, true
}

func writeCache(path string, data cacheData) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func cacheExpired(data cacheData, now time.Time, ttl time.Duration) bool {
	if data.CheckedAt == "" {
		return true
	}
	checkedAt, err := time.Parse(time.RFC3339Nano, data.CheckedAt)
	if err != nil {
		return true
	}
	return now.Sub(checkedAt) > ttl
}

func fetchLatestRelease(ctx context.Context, opts Options) (string, error) {
	url := strings.TrimSpace(opts.UpdateURL)
	if url == "" {
		url = DefaultUpdateURL
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "small-updatecheck")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release releaseResponse
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}
	return strings.TrimSpace(release.TagName), nil
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		return "v" + strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	}
	return "v" + v
}

type semver struct {
	major int
	minor int
	patch int
	pre   bool
	valid bool
}

func parseSemver(v string) semver {
	v = strings.TrimSpace(v)
	if v == "" {
		return semver{}
	}
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	main := v
	if idx := strings.Index(v, "-"); idx >= 0 {
		main = v[:idx]
	}
	parts := strings.Split(main, ".")
	if len(parts) == 0 {
		return semver{}
	}
	toInt := func(s string) (int, bool) {
		if s == "" {
			return 0, true
		}
		val, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return val, true
	}
	major, ok := toInt(parts[0])
	if !ok {
		return semver{}
	}
	minor := 0
	patch := 0
	if len(parts) > 1 {
		val, ok := toInt(parts[1])
		if !ok {
			return semver{}
		}
		minor = val
	}
	if len(parts) > 2 {
		val, ok := toInt(parts[2])
		if !ok {
			return semver{}
		}
		patch = val
	}
	pre := strings.Contains(v, "-")
	return semver{major: major, minor: minor, patch: patch, pre: pre, valid: true}
}

func compareSemver(a, b semver) int {
	if !a.valid || !b.valid {
		return 0
	}
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	if a.pre != b.pre {
		if a.pre {
			return -1
		}
		return 1
	}
	return 0
}

func isNewerVersion(latest, current string) bool {
	latestParsed := parseSemver(latest)
	currentParsed := parseSemver(current)
	return compareSemver(latestParsed, currentParsed) > 0
}
