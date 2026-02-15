package main

import "syscall"

func SysClose(fd int) error {
	return syscall.CloseHandle(syscall.Handle(fd))
}
