package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startProcess(binPath string, args []string) *exec.Cmd {
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // own process group, survives manager restart
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("start %s: %v", binPath, err)
	}
	log.Printf("started %s PID=%d", binPath, cmd.Process.Pid)
	return cmd
}

func monitorPID(pidFile string, pid int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !isAlive(pid) {
			return
		}
		// Check if PID file changed (another upgrade happened).
		if cur := readPID(pidFile); cur > 0 && cur != pid {
			log.Printf("PID file changed %d → %d, switching monitor", pid, cur)
			pid = cur
		}
	}
}
