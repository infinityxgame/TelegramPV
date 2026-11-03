//go:build !darwin

package updater

func (u *AppUpdater) installPlatform(_, _ string) (bool, error) {
	return false, nil
}

func (u *AppUpdater) cleanPlatform() {}
