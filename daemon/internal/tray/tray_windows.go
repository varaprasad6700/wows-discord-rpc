//go:build windows

package tray

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW  = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procCreatePopupMenu    = user32.NewProc("CreatePopupMenu")
	procAppendMenuW        = user32.NewProc("AppendMenuW")
	procTrackPopupMenu     = user32.NewProc("TrackPopupMenu")
	procGetCursorPos       = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procDestroyMenu        = user32.NewProc("DestroyMenu")
	procLoadIconW          = user32.NewProc("LoadIconW")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procShellNotifyIconW   = shell32.NewProc("Shell_NotifyIconW")
)

const (
	WM_USER         = 0x0400
	WM_TRAYICON     = WM_USER + 1
	WM_COMMAND      = 0x0111
	WM_RBUTTONUP    = 0x0205
	WM_LBUTTONDBLCLK = 0x0203

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800
	MF_DISABLED  = 0x00000002
	MF_GRAYED    = 0x00000001

	TPM_BOTTOMALIGN = 0x0020
	TPM_LEFTALIGN   = 0x0000

	ID_STATUS_HEADER = 1001
	ID_OPEN_UI       = 1002
	ID_VIEW_LOGS     = 1003
	ID_CHECK_UPDATES = 1004
	ID_EXIT          = 1005

	IDI_APPLICATION = 32512
)

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
}

type POINT struct {
	X int32
	Y int32
}

type TrayManager struct {
	hwnd           uintptr
	nid            NOTIFYICONDATA
	onOpenUI       func()
	onOpenLogs     func()
	onCheckUpdates func()
	onExit         func()
}

var currentTray *TrayManager

// LaunchAppMode opens the controller in standalone Edge application window mode
func LaunchAppMode(url string) error {
	edgePaths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}
	var edgeExe string
	for _, p := range edgePaths {
		if _, err := os.Stat(p); err == nil {
			edgeExe = p
			break
		}
	}
	if edgeExe == "" {
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}

	appArg := fmt.Sprintf("--app=%s", url)
	return exec.Command(edgeExe, appArg, "--window-size=540,680").Start()
}

func Remove() {
	if currentTray != nil && currentTray.hwnd != 0 {
		procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&currentTray.nid)))
		currentTray.hwnd = 0
	}
}

func Run(onOpenUI, onOpenLogs, onCheckUpdates, onExit func()) error {
	className, _ := syscall.UTF16PtrFromString("WoWSPresenceTrayClass")
	windowName, _ := syscall.UTF16PtrFromString("WoWSPresenceTrayWindow")

	wndProc := syscall.NewCallback(func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case WM_TRAYICON:
			switch lParam {
			case WM_LBUTTONDBLCLK:
				if currentTray != nil && currentTray.onOpenUI != nil {
					currentTray.onOpenUI()
				}
			case WM_RBUTTONUP:
				showContextMenu(hwnd)
			}
			return 0

		case WM_COMMAND:
			cmdId := uint32(wParam & 0xFFFF)
			if currentTray != nil {
				switch cmdId {
				case ID_OPEN_UI:
					if currentTray.onOpenUI != nil {
						currentTray.onOpenUI()
					}
				case ID_VIEW_LOGS:
					if currentTray.onOpenLogs != nil {
						currentTray.onOpenLogs()
					}
				case ID_CHECK_UPDATES:
					if currentTray.onCheckUpdates != nil {
						currentTray.onCheckUpdates()
					}
				case ID_EXIT:
					if currentTray.onExit != nil {
						currentTray.onExit()
					}
					procPostQuitMessage.Call(0)
				}
			}
			return 0
		}

		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	})

	type WNDCLASSEX struct {
		CbSize        uint32
		Style         uint32
		LpfnWndProc   uintptr
		CbClsExtra    int32
		CbWndExtra    int32
		HInstance     uintptr
		HIcon         uintptr
		HCursor       uintptr
		HbrBackground uintptr
		LpszMenuName  *uint16
		LpszClassName *uint16
		HIconSm       uintptr
	}

	var hIcon uintptr
	hIcon, _, _ = procLoadIconW.Call(0, uintptr(IDI_APPLICATION))

	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   wndProc,
		LpszClassName: className,
		HIcon:         hIcon,
		HIconSm:       hIcon,
	}

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0, 0, 0, 0, 0,
		0, 0, 0, 0,
	)

	if hwnd == 0 {
		return fmt.Errorf("failed to create tray window")
	}

	currentTray = &TrayManager{
		hwnd:           hwnd,
		onOpenUI:       onOpenUI,
		onOpenLogs:     onOpenLogs,
		onCheckUpdates: onCheckUpdates,
		onExit:         onExit,
	}

	// Add Tray Icon
	currentTray.nid.CbSize = uint32(unsafe.Sizeof(currentTray.nid))
	currentTray.nid.HWnd = hwnd
	currentTray.nid.UID = 1
	currentTray.nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	currentTray.nid.UCallbackMessage = WM_TRAYICON
	currentTray.nid.HIcon = hIcon

	tip, _ := syscall.UTF16FromString("World of Warships Presence")
	copy(currentTray.nid.SzTip[:], tip)

	procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&currentTray.nid)))

	// Run message pump
	type MSG struct {
		HWnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      POINT
	}

	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	// Cleanup tray icon on exit
	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&currentTray.nid)))
	return nil
}

func showContextMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	titleText, _ := syscall.UTF16PtrFromString("● WoWS Presence")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING|MF_DISABLED|MF_GRAYED), uintptr(ID_STATUS_HEADER), uintptr(unsafe.Pointer(titleText)))

	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)

	openText, _ := syscall.UTF16PtrFromString("Open Controller UI")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_OPEN_UI), uintptr(unsafe.Pointer(openText)))

	logsText, _ := syscall.UTF16PtrFromString("View Log File")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_VIEW_LOGS), uintptr(unsafe.Pointer(logsText)))

	updateText, _ := syscall.UTF16PtrFromString("Check for Updates...")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_CHECK_UPDATES), uintptr(unsafe.Pointer(updateText)))

	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)

	exitText, _ := syscall.UTF16PtrFromString("Exit Presence")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_EXIT), uintptr(unsafe.Pointer(exitText)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(hwnd)
	procTrackPopupMenu.Call(hMenu, uintptr(TPM_BOTTOMALIGN|TPM_LEFTALIGN), uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
}
