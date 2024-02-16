package service

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// getProcessArgs returns the operating system specific arguments that
// are needed to "detach" the child process from this parent process in
// which the go program is running.
//
// These properties don't detach a child "correctly".
// The correct way would be using the flag "windows.DETACHED_PROCESS".
// But with this one it is impossible to not open a command prompt (even with "NO_WINDOW").
// So you have to use a constaletation with "START" and "CALL" scripts to detach the running process
func (e *ProgramExecutor) getProcessArgs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// Run process in background
		CreationFlags: windows.CREATE_NO_WINDOW, // windows.DETACHED_PROCESS, // syscall.CREATE_NEW_PROCESS_GROUP

		// We don't show a CMD window by default. If the user does want a CLI windows,
		// he would have to write a batch script that opens up a new cmd process
		HideWindow: true,
	}
}

// startProgramm executes the given program with the provided arguments in the
// background with operating system specific arguments that are needed to
// "detach" the child process from his parent process.
//
// This method does not block or wait until the program was executed
func (e *ProgramExecutor) startProgramm(program string, args []string) error {

	// Wrap the main command with call and start scripts
	wrapped := []string{"/Q", "/C", "CALL", "START", "/B", program}
	wrapped = append(wrapped, args...)

	// Call it
	cmd := exec.Command("cmd.exe", wrapped...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = e.getProcessArgs()

	return cmd.Start()
}
