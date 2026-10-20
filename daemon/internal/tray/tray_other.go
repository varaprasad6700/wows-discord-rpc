//go:build !windows

package tray

func LaunchAppMode(url string) error {
	return nil
}

func Run(onOpenUI, onOpenLogs, onCheckUpdates, onExit func()) error {
	return nil
}
