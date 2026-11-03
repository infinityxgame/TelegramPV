package updater

// Comprobación e instalación de actualizaciones.
//
// No se usan librerías externas de actualización para mantener el binario ligero.
// Se realizan llamadas a la API de GitHub Releases, comparación de semver
// e intercambio atómico del ejecutable usando la biblioteca estándar.

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
)

type Progress struct {
	Status     string `json:"status"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
	Percentage int    `json:"percentage"`
}

type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

type ReleaseInfo struct {
	TagName string         `json:"tag_name"`
	Body    string         `json:"body"`
	Assets  []ReleaseAsset `json:"assets"`

	// asset es el archivo que corresponde a este sistema, ya elegido durante la comprobación.
	asset ReleaseAsset
}

type AppUpdater struct {
	currentVersion  string
	repoURL         string
	mu              sync.RWMutex
	progress        Progress
	postponedUpdate string

	checkCount       int
	announcedVersion string
	lastCheckError   string

	rateLimitUntil time.Time
}

var errRateLimited = errors.New("GitHub rate limited update requests")

const (
	githubAPI     = "https://api.github.com"
	noDataTimeout = 60 * time.Second
)

var updaterTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   15 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ForceAttemptHTTP2:     true,
}

var apiClient = &http.Client{
	Transport: updaterTransport,
	Timeout:   30 * time.Second,
}

var downloadClient = &http.Client{Transport: updaterTransport}

func userAgent() string {
	return "TelegramPV/" + config.AppVersion + " (+https://github.com/" + config.GithubRepo + ")"
}

func NewAppUpdater() *AppUpdater {
	config.InitPaths()
	u := &AppUpdater{
		currentVersion: config.AppVersion,
		repoURL:        config.GithubRepo,
		progress: Progress{
			Status: "idle",
		},
	}
	u.CleanOldVersion()
	return u
}

// CleanOldVersion busca y elimina archivos temporales de actualizaciones previas (.old)
func (u *AppUpdater) CleanOldVersion() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	dir := filepath.Dir(exePath)
	base := filepath.Base(exePath)

	targets := []string{
		exePath + ".old",
		filepath.Join(dir, "."+base+".old"),
	}

	for _, target := range targets {
		if _, err := os.Stat(target); err == nil {
			log.Printf("[UPDATER] %s", i18n.T("updater.oldFileFound", target))
			if err := os.Remove(target); err != nil {
				log.Printf("[UPDATER] %s", i18n.T("updater.oldFileError", target, err))
			} else {
				log.Printf("[UPDATER] %s", i18n.T("updater.oldFileRemoved", target))
			}
		}
	}

	u.cleanPlatform()
}

func (u *AppUpdater) GetProgress() Progress {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.progress
}

func (u *AppUpdater) setProgress(status string, downloaded, total int64, pct int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.progress = Progress{
		Status:     status,
		Downloaded: downloaded,
		Total:      total,
		Percentage: pct,
	}
}

func (u *AppUpdater) Postpone(version string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.postponedUpdate = version
}

func (u *AppUpdater) IsPostponed(version string) bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.postponedUpdate == version
}

func (u *AppUpdater) routineLog(first bool, message, detail string) {
	if first {
		logbus.Info(logbus.CatUpdater, message, detail)
	} else {
		logbus.Info(logbus.CatUpdater, message, detail)
	}
}

func (u *AppUpdater) checkErrorLog(first bool, message string, err error) {
	text := ""
	if err != nil {
		text = err.Error()
	}

	u.mu.Lock()
	changed := u.lastCheckError != text
	u.lastCheckError = text
	u.mu.Unlock()

	if first || changed {
		logbus.Warn(logbus.CatUpdater, message, text)
		return
	}
	logbus.Warn(logbus.CatUpdater, message, text)
}

type releaseGitHub struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Body       string        `json:"body"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []assetGitHub `json:"assets"`
}

type assetGitHub struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

func (u *AppUpdater) recordRateLimit(resp *http.Response) bool {
	retryAfter := resp.Header.Get("Retry-After")
	exhausted := resp.Header.Get("X-RateLimit-Remaining") == "0"
	if !exhausted && retryAfter == "" {
		return false
	}

	wait := 15 * time.Minute
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
			wait = time.Duration(seconds) * time.Second
		}
	} else if reset := strings.TrimSpace(resp.Header.Get("X-RateLimit-Reset")); reset != "" {
		if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if remaining := time.Until(time.Unix(epoch, 0)); remaining > 0 {
				wait = remaining
			}
		}
	}
	wait += 30 * time.Second

	u.mu.Lock()
	u.rateLimitUntil = time.Now().Add(wait)
	u.mu.Unlock()

	logbus.Warn(logbus.CatUpdater,
		i18n.T("updater.rateLimited"),
		i18n.T("updater.rateLimitedDetail", wait.Round(time.Second)))
	return true
}

func (u *AppUpdater) fetchLatestRelease(ctx context.Context) (*releaseGitHub, error) {
	slug := strings.Trim(strings.TrimSpace(u.repoURL), "/")
	if slug == "" || !strings.Contains(slug, "/") {
		return nil, errors.New(i18n.T("updater.repoMisconfigured", u.repoURL))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, slug), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", userAgent())

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, nil
	case http.StatusForbidden, http.StatusTooManyRequests:
		if u.recordRateLimit(resp) {
			return nil, errRateLimited
		}
		return nil, errors.New(i18n.T("updater.githubRejected", resp.StatusCode))
	default:
		return nil, errors.New(i18n.T("updater.githubResponseCode", resp.StatusCode))
	}

	var rel releaseGitHub
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return nil, errors.New(i18n.T("updater.githubReadError", err))
	}
	return &rel, nil
}

var nonExecutableExtensions = map[string]bool{
	".txt": true, ".md": true, ".sig": true, ".minisig": true,
	".asc": true, ".sha256": true, ".sha512": true, ".pem": true,
}

func systemKeywords() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"windows", "win", ".exe"}
	case "darwin":
		return []string{"darwin", "macos", "mac", "osx"}
	case "linux":
		return []string{"linux"}
	default:
		return []string{runtime.GOOS}
	}
}

func archKeywords() []string {
	switch runtime.GOARCH {
	case "amd64":
		return []string{"amd64", "x86_64", "x64"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	case "386":
		return []string{"386", "i386", "x86"}
	default:
		return []string{runtime.GOARCH}
	}
}

func containsAnyKeyword(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func assetForThisSystem(assets []assetGitHub) *ReleaseAsset {
	system := systemKeywords()
	architecture := archKeywords()

	var candidate *assetGitHub
	for i := range assets {
		asset := &assets[i]
		if asset.URL == "" {
			continue
		}
		name := strings.ToLower(asset.Name)
		if nonExecutableExtensions[strings.ToLower(filepath.Ext(name))] {
			continue
		}
		if !containsAnyKeyword(name, system) {
			continue
		}
		if containsAnyKeyword(name, architecture) {
			candidate = asset
			break
		}
		if candidate == nil {
			candidate = asset
		}
	}

	if candidate == nil {
		return nil
	}
	return &ReleaseAsset{
		Name:        candidate.Name,
		DownloadURL: candidate.URL,
		Size:        candidate.Size,
	}
}

func numericPrefix(s string) int {
	fin := 0
	for fin < len(s) && s[fin] >= '0' && s[fin] <= '9' {
		fin++
	}
	if fin == 0 {
		return 0
	}
	n, err := strconv.Atoi(s[:fin])
	if err != nil {
		return 0
	}
	return n
}

func normalizeVersion(v string) (numbers [3]int, preRelease string) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	if i := strings.IndexByte(v, '-'); i >= 0 {
		preRelease = v[i+1:]
		v = v[:i]
	}

	for i, part := range strings.SplitN(v, ".", 3) {
		numbers[i] = numericPrefix(strings.TrimSpace(part))
	}
	return numbers, preRelease
}

func compareVersions(a, b string) int {
	na, preReleaseA := normalizeVersion(a)
	nb, preReleaseB := normalizeVersion(b)

	for i := 0; i < len(na); i++ {
		if na[i] != nb[i] {
			if na[i] < nb[i] {
				return -1
			}
			return 1
		}
	}

	switch {
	case preReleaseA == "" && preReleaseB == "":
		return 0
	case preReleaseA == "":
		return 1
	case preReleaseB == "":
		return -1
	}
	return strings.Compare(preReleaseA, preReleaseB)
}

func (u *AppUpdater) CheckForUpdate() (*ReleaseInfo, *ReleaseAsset, error) {
	u.mu.Lock()
	u.checkCount++
	first := u.checkCount == 1
	pausedUntil := u.rateLimitUntil
	u.mu.Unlock()

	if !pausedUntil.IsZero() && time.Now().Before(pausedUntil) {
		logbus.Info(logbus.CatUpdater, i18n.T("updater.checkPaused"),
			i18n.T("updater.checkPausedDetail", time.Until(pausedUntil).Round(time.Second)))
		return nil, nil, nil
	}

	u.routineLog(first, i18n.T("updater.checking", u.repoURL),
		i18n.T("updater.checkingDetail", u.currentVersion))

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	rel, err := u.fetchLatestRelease(ctx)
	if err != nil {
		if !errors.Is(err, errRateLimited) {
			u.checkErrorLog(first, i18n.T("updater.checkError"), err)
		}
		return nil, nil, err
	}
	if rel == nil || rel.Draft || strings.TrimSpace(rel.TagName) == "" {
		u.routineLog(first, i18n.T("updater.noReleases"), "")
		return nil, nil, nil
	}

	asset := assetForThisSystem(rel.Assets)
	if asset == nil {
		u.routineLog(first, i18n.T("updater.noAsset"),
			i18n.T("updater.noAssetDetail", rel.TagName, len(rel.Assets)))
		return nil, nil, nil
	}

	u.routineLog(first, i18n.T("updater.latestRelease", rel.TagName),
		i18n.T("updater.latestReleaseDetail", asset.Name))

	if compareVersions(rel.TagName, u.currentVersion) <= 0 {
		u.routineLog(first, i18n.T("updater.upToDate", u.currentVersion), "")
		return nil, nil, nil
	}

	u.mu.Lock()
	alreadyAnnounced := u.announcedVersion == rel.TagName
	u.announcedVersion = rel.TagName
	u.mu.Unlock()

	if alreadyAnnounced {
		logbus.Info(logbus.CatUpdater,
			i18n.T("updater.stillAvailable", rel.TagName), "")
	} else {
		logbus.Success(logbus.CatUpdater,
			i18n.T("updater.newVersion", rel.TagName),
			i18n.T("updater.newVersionDetail", u.currentVersion))
	}

	info := &ReleaseInfo{
		TagName: rel.TagName,
		Body:    rel.Body,
		Assets:  []ReleaseAsset{*asset},
		asset:   *asset,
	}
	return info, &info.Assets[0], nil
}

func (u *AppUpdater) InstallUpdate(rel *ReleaseInfo) error {
	if rel == nil || strings.TrimSpace(rel.asset.DownloadURL) == "" {
		return errors.New(i18n.T("updater.updateInfoInvalid"))
	}
	asset := rel.asset

	go func() {
		u.setProgress("starting", 0, 0, 0)

		tempDir, err := os.MkdirTemp("", "telegrampv_update")
		if err != nil {
			u.setProgress("error: "+i18n.T("updater.tempDirError"), 0, 0, 0)
			return
		}
		defer os.RemoveAll(tempDir)

		archivePath := filepath.Join(tempDir, filepath.Base(asset.Name))
		u.setProgress("downloading", 0, asset.Size, 0)

		if err := u.downloadWithProgress(asset.DownloadURL, archivePath); err != nil {
			u.setProgress(i18n.T("updater.downloadFailedPrefix")+err.Error(), 0, 0, 0)
			return
		}

		u.setProgress("verifying", 0, 0, 100)
		if err := verifyChecksum(archivePath, asset.DownloadURL, asset.Name); err != nil {
			u.setProgress("error: "+err.Error(), 0, 0, 0)
			logbus.Error(logbus.CatUpdater, i18n.T("updater.verifyRejected"), err.Error())
			return
		}

		if handled, errPlat := u.installPlatform(archivePath, tempDir); handled {
			if errPlat != nil {
				u.setProgress("error: "+errPlat.Error(), 0, 0, 0)
				logbus.Error(logbus.CatUpdater, i18n.T("updater.applyError"), errPlat.Error())
			}
			return
		}

		u.setProgress("extracting", 0, 0, 100)
		extractPath := filepath.Join(tempDir, "extracted")
		_ = os.MkdirAll(extractPath, 0755)

		lowerArch := strings.ToLower(archivePath)
		if strings.HasSuffix(lowerArch, ".zip") {
			if err := unzip(archivePath, extractPath); err != nil {
				u.setProgress(i18n.T("updater.unzipFailedPrefix")+err.Error(), 0, 0, 0)
				return
			}
		} else if strings.HasSuffix(lowerArch, ".tar.gz") || strings.HasSuffix(lowerArch, ".tgz") {
			if err := untarGzip(archivePath, extractPath); err != nil {
				u.setProgress(i18n.T("updater.untarFailedPrefix")+err.Error(), 0, 0, 0)
				return
			}
		} else {
			extractPath = archivePath
		}

		newBinaryPath := u.findExecutable(extractPath)
		if newBinaryPath == "" {
			u.setProgress(i18n.T("updater.exeNotFound"), 0, 0, 0)
			return
		}

		u.setProgress("finishing", 0, 0, 100)
		exePath, err := os.Executable()
		if err != nil {
			u.setProgress("error: "+i18n.T("updater.exePathError"), 0, 0, 0)
			return
		}

		log.Printf("[UPDATER] %s", i18n.T("updater.applying", exePath))
		if err := applyBinary(newBinaryPath, exePath); err != nil {
			u.setProgress(i18n.T("updater.applyFailedPrefix")+err.Error(), 0, 0, 0)
			logbus.Error(logbus.CatUpdater, i18n.T("updater.applyError"), err.Error())
			return
		}

		logbus.Success(logbus.CatUpdater, i18n.T("updater.applied"), exePath)
		u.setProgress("finishing", 0, 0, 100)
		time.Sleep(2 * time.Second)

		u.restartApp(exePath)
	}()

	return nil
}

func applyBinary(source, target string) error {
	perms := os.FileMode(0o755)
	if fi, err := os.Stat(target); err == nil {
		if p := fi.Mode().Perm(); p&0o100 != 0 {
			perms = p
		}
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return errors.New(i18n.T("updater.readNewVersionError", err))
	}
	defer sourceFile.Close()

	tmpFile, err := os.CreateTemp(filepath.Dir(target), ".telegrampv-new-*")
	if err != nil {
		return errors.New(i18n.T("updater.writeError", filepath.Dir(target), err))
	}
	tempPath := tmpFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(tmpFile, sourceFile); err != nil {
		_ = tmpFile.Close()
		return errors.New(i18n.T("updater.copyError", err))
	}
	_ = tmpFile.Sync()
	if err := tmpFile.Close(); err != nil {
		return errors.New(i18n.T("updater.closeError", err))
	}
	if err := os.Chmod(tempPath, perms); err != nil {
		return errors.New(i18n.T("updater.chmodError", err))
	}

	oldFile := target + ".old"
	_ = os.Remove(oldFile)
	if err := os.Rename(target, oldFile); err != nil {
		return errors.New(i18n.T("updater.moveOldError", target, err))
	}

	if err := os.Rename(tempPath, target); err != nil {
		if errRollback := os.Rename(oldFile, target); errRollback != nil {
			return errors.New(i18n.T("updater.rollbackFailed", err, oldFile, errRollback))
		}
		return errors.New(i18n.T("updater.putNewError", err))
	}
	removeTemp = false

	if err := os.Remove(oldFile); err != nil {
		log.Printf("[UPDATER] %s", i18n.T("updater.cleanupNextBoot", err))
	}
	return nil
}

func (u *AppUpdater) downloadWithProgress(downloadURL, dest string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchdog := time.AfterFunc(noDataTimeout, cancel)
	defer watchdog.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(i18n.T("updater.githubStatusCode", resp.StatusCode))
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	buffer := make([]byte, 64*1024)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			watchdog.Reset(noDataTimeout)
			if _, werr := out.Write(buffer[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			pct := 0
			if total > 0 {
				pct = int(float64(downloaded) / float64(total) * 100)
			}
			u.setProgress("downloading", downloaded, total, pct)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				return errors.New(i18n.T("updater.downloadTimeout", noDataTimeout))
			}
			return err
		}
	}

	if total > 0 && downloaded != total {
		return errors.New(i18n.T("updater.downloadIncomplete", downloaded, total))
	}
	return nil
}

func (u *AppUpdater) findExecutable(path string) string {
	fi, err := os.Stat(path)
	if err == nil && !fi.IsDir() {
		return path
	}

	var found string
	var priority int = -1

	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		name := strings.ToLower(info.Name())
		ext := strings.ToLower(filepath.Ext(name))

		isWindows := runtime.GOOS == "windows"
		if isWindows && ext != ".exe" {
			return nil
		}

		currPriority := -1
		if name == "telegrampv.exe" || name == "tgpv.exe" || name == "telegrampv" || name == "tgpv" {
			currPriority = 100
		} else if strings.Contains(name, "telegrampv") || strings.Contains(name, "tgpv") {
			currPriority = 50
		} else if !isWindows && ext == "" {
			currPriority = 10
		}

		if currPriority > priority {
			priority = currPriority
			found = p
		}

		if priority == 100 {
			return io.EOF
		}
		return nil
	})
	return found
}

func (u *AppUpdater) restartApp(exePath string) {
	cmd := exec.Command(exePath, os.Args[1:]...)

	log.Printf("[UPDATER] %s", i18n.T("updater.relaunching", exePath))
	err := cmd.Start()
	if err != nil {
		log.Printf("[UPDATER] %s", i18n.T("updater.relaunchError", err))
	}

	os.Exit(0)
}

func isWithin(dest, target string) bool {
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	if targetAbs == destAbs {
		return true
	}
	return strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator))
}

func safeMode(mode int64) os.FileMode {
	perm := os.FileMode(mode).Perm()
	if perm == 0 {
		perm = 0o644
	}
	if perm&0o111 != 0 {
		perm |= 0o100
	}
	return perm
}

const checksumsFileName = "checksums.txt"

func verifyChecksum(archivePath, assetURL, assetName string) error {
	expected, err := downloadChecksum(assetURL, assetName)
	if err != nil {
		return err
	}

	actual, err := sha256OfFile(archivePath)
	if err != nil {
		return errors.New(i18n.T("updater.signatureError", err))
	}

	if !strings.EqualFold(actual, expected) {
		return errors.New(i18n.T("updater.signatureMismatch", expected, actual))
	}

	logbus.Info(logbus.CatUpdater, i18n.T("updater.verified"), assetName)
	return nil
}

func sha256OfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadChecksum(assetURL, assetName string) (string, error) {
	checksumURL, err := checksumURL(assetURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", errors.New(i18n.T("updater.checksumDownloadError", checksumsFileName, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(i18n.T("updater.checksumNotPublished", checksumsFileName, resp.StatusCode))
	}

	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) == filepath.Base(assetName) {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", errors.New(i18n.T("updater.checksumReadError", checksumsFileName, err))
	}

	return "", errors.New(i18n.T("updater.checksumMissingEntry", checksumsFileName, assetName))
}

func checksumURL(assetURL string) (string, error) {
	u, err := url.Parse(assetURL)
	if err != nil {
		return "", errors.New(i18n.T("updater.invalidDownloadURL", err))
	}
	i := strings.LastIndex(u.Path, "/")
	if i < 0 {
		return "", errors.New(i18n.T("updater.invalidDownloadURLText", assetURL))
	}
	u.Path = u.Path[:i+1] + checksumsFileName
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !isWithin(dest, fpath) {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, 0o755)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, safeMode(int64(f.Mode())))
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func untarGzip(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		if !isWithin(dest, target) {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0o755)
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, safeMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, tr)
			outFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
