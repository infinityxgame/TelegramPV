//go:build darwin

package updater

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
)

const appTranslocationMarker = "/AppTranslocation/"

func (u *AppUpdater) installPlatform(archivePath, tempDir string) (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, nil
	}
	if resolved, errLink := filepath.EvalSymlinks(exePath); errLink == nil {
		exePath = resolved
	}

	currentBundle := bundleForExecutable(exePath)
	if currentBundle == "" {
		return false, nil
	}

	if strings.Contains(currentBundle, appTranslocationMarker) {
		return true, errors.New(i18n.T("updater.darwinTranslocation"))
	}

	stage := filepath.Join(tempDir, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return true, errors.New(i18n.T("updater.darwinTempDirError", err))
	}
	if err := extractForMac(archivePath, stage); err != nil {
		return true, err
	}

	newBundle := findAppBundle(stage)
	if newBundle == "" {
		return true, errors.New(i18n.T("updater.darwinNoAppBundle"))
	}

	_ = exec.Command("xattr", "-dr", "com.apple.quarantine", newBundle).Run()

	backup := currentBundle + ".old"
	_ = os.RemoveAll(backup)
	if err := os.Rename(currentBundle, backup); err != nil {
		return true, errors.New(i18n.T("updater.darwinReplaceError", currentBundle, err))
	}

	if err := runTool("ditto", newBundle, currentBundle); err != nil {
		_ = os.RemoveAll(currentBundle)
		if rollbackErr := os.Rename(backup, currentBundle); rollbackErr != nil {
			return true, errors.New(i18n.T("updater.darwinRollbackFailed", err, rollbackErr))
		}
		return true, errors.New(i18n.T("updater.darwinCopyError", err))
	}

	if err := os.RemoveAll(backup); err != nil {
		log.Printf("[UPDATER] %s", i18n.T("updater.oldFileError", backup, err))
	}

	logbus.Success(logbus.CatUpdater, i18n.T("updater.applied"), currentBundle)
	u.setProgress("finishing", 0, 0, 100)

	relaunchBundle(currentBundle)

	time.Sleep(1 * time.Second)
	_ = os.RemoveAll(tempDir)
	os.Exit(0)

	return true, nil
}

func (u *AppUpdater) cleanPlatform() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, errLink := filepath.EvalSymlinks(exePath); errLink == nil {
		exePath = resolved
	}

	bundle := bundleForExecutable(exePath)
	if bundle == "" {
		return
	}

	backup := bundle + ".old"
	if _, err := os.Stat(backup); err != nil {
		return
	}
	log.Printf("[UPDATER] %s", i18n.T("updater.oldBundleFound", backup))
	if err := os.RemoveAll(backup); err != nil {
		log.Printf("[UPDATER] %s", i18n.T("updater.oldBundleError", backup, err))
	}
}

func bundleForExecutable(exePath string) string {
	macOSDir := filepath.Dir(exePath)
	contentsDir := filepath.Dir(macOSDir)
	bundle := filepath.Dir(contentsDir)

	if filepath.Base(macOSDir) != "MacOS" || filepath.Base(contentsDir) != "Contents" {
		return ""
	}
	if !strings.HasSuffix(bundle, ".app") {
		return ""
	}
	return bundle
}

func findAppBundle(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "__MACOSX" {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(d.Name(), ".app") {
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(path, "Contents", "MacOS")); statErr != nil {
			return nil
		}
		found = path
		return filepath.SkipAll
	})
	return found
}

func extractForMac(archivePath, dest string) error {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		if err := runTool("ditto", "-x", "-k", archivePath, dest); err != nil {
			return errors.New(i18n.T("updater.darwinExtractError", err))
		}
		return nil
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return untarGzip(archivePath, dest)
	default:
		return errors.New(i18n.T("updater.darwinUnsupportedFormat", filepath.Base(archivePath)))
	}
}

func runTool(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err, detail)
	}
	return nil
}

func relaunchBundle(bundle string) {
	script := fmt.Sprintf("sleep 2; open -n %s", shellQuote(bundle))
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	log.Printf("[UPDATER] %s", i18n.T("updater.relaunching", bundle))
	if err := cmd.Start(); err != nil {
		log.Printf("[UPDATER] %s", i18n.T("updater.relaunchError", err))
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
