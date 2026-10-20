package process

import (
	"strings"
	"syscall"
	"unsafe"
)

const TH32CS_SNAPPROCESS = 0x00000002

type PROCESSENTRY32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [syscall.MAX_PATH]uint16
}

func IsProcessRunning(name string) bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createToolhelp32Snapshot := kernel32.NewProc("CreateToolhelp32Snapshot")
	process32First := kernel32.NewProc("Process32FirstW")
	process32Next := kernel32.NewProc("Process32NextW")

	snapshot, _, _ := createToolhelp32Snapshot.Call(uintptr(TH32CS_SNAPPROCESS), 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(snapshot))

	var entry PROCESSENTRY32
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := process32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	targetName := strings.ToLower(name)

	for ret != 0 {
		exeName := syscall.UTF16ToString(entry.ExeFile[:])
		if strings.ToLower(exeName) == targetName {
			return true
		}
		ret, _, _ = process32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}

	return false
}

func IsGameRunning() bool {
	return IsProcessRunning("WorldOfWarships64.exe") || IsProcessRunning("WorldOfWarships.exe")
}
