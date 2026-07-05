// gofr-manager is a keep-alive process manager for gofr applications using
// fd-inheritance graceful restart.
//
// It starts the target binary, monitors its PID file, and restarts on crash.
// When the application performs a graceful restart via SIGHUP (fd inheritance),
// the manager automatically tracks the new process via the PID file.
//
// Usage:
//
//	gofr-manager /path/to/app [args...]
//
// The application must write its PID to a file via app.WritePID().
// The manager reads ./pid by default, or checks the PID_FILE environment variable.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gofr-manager /path/to/app [args...]\n")
		os.Exit(1)
	}

	binPath := os.Args[1]
	var args []string
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	pidFile := os.Getenv("PID_FILE")
	if pidFile == "" {
		pidFile = "./pid"
	}

	// Forward SIGHUP to the managed process (for graceful restart).
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP)
		for range sig {
			pid := readPID(pidFile)
			if pid > 0 {
				p, err := os.FindProcess(pid)
				if err == nil {
					p.Signal(syscall.SIGHUP)
				}
			}
		}
	}()

	// Main supervision loop.
	var cmd *exec.Cmd
	cmd = startProcess(binPath, args)

	for {
		// Wait for the child to exit.
		err := cmd.Wait()
		exitPID := cmd.Process.Pid
		log.Printf("process PID=%d exited: %v", exitPID, err)

		// Check if a new process took over (fd inheritance handoff).
		curPID := readPID(pidFile)
		if curPID > 0 && curPID != exitPID {
			log.Printf("detected upgraded process PID=%d via %s, monitoring", curPID, pidFile)
			monitorPID(pidFile, curPID)
			// monitorPID returns when the process is dead — restart.
			log.Printf("upgraded process PID=%d died, restarting", curPID)
		} else {
			log.Println("no upgraded process found, restarting")
		}

		// Avoid tight crash loop.
		time.Sleep(500 * time.Millisecond)
		cmd = startProcess(binPath, args)
	}
}
