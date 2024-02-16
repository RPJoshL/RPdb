//go:build unix

package service

import (
	"fmt"
	"os"
	"syscall"
)

// getProcessArgs returns the operating system specific arguments that
// are needed to "detach" the child process from this parent process in
// which the go program is running
func (e *ProgramExecutor) getProcessArgs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// startProgramm executes the given program with the provided arguments in the
// background with operating system specific arguments that are needed to
// "detach" the child process from his parent process.
//
// This method does not block or wait until the program was executed
func (e *ProgramExecutor) startProgramm(program string, args []string) error {

	// os.StartProcess passes the args raw → include also the program name
	rtc := []string{program}
	if len(args) > 0 {
		rtc = append(rtc, args...)
	}

	// This method (forking) does only work for unix systems
	process, err := os.StartProcess(program, rtc, &os.ProcAttr{
		Env: os.Environ(),
		Sys: e.getProcessArgs(),
	})

	if err != nil {
		return err
	} else {
		// Detach process
		if err := process.Release(); err != nil {
			return fmt.Errorf("failed to detach process: %s", err)
		}
	}

	return nil
}
