//go:build !windows

package tray

func LaunchAppMode(url string) error {
	return nil
}

func Remove() {}

func Run(onOpenUI, onOpenLogs, onCheckUpdates, onExit func()) error {
	return nil
}
