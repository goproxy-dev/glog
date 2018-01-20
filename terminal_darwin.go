// +build darwin

package glog

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	colorRed    = []byte{'\x1b', '[', '3', '1', 'm'}
	colorYellow = []byte{'\x1b', '[', '3', '3', 'm'}
	colorReset  = []byte{'\x1b', '[', '0', 'm'}
)

func WriteFileWithColor(file *os.File, data []byte, s severity) {
	switch s {
	case fatalLog:
		file.Write(colorRed)
		file.Write(data)
		file.Write(colorReset)
	case errorLog:
		file.Write(colorRed)
		file.Write(data)
		file.Write(colorReset)
	case warningLog:
		file.Write(colorYellow)
		file.Write(data)
		file.Write(colorReset)
	case infoLog:
		file.Write(data)
	}
}

func IsTerminal(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCGETA, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func RedirectStderrTo(file *os.File) error {
	os.Stderr = file
	return syscall.Dup2(int(file.Fd()), 2)
}
