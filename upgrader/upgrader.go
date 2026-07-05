// Package upgrader provides fd-inheritance based graceful restart.
//
// Usage:
//
//	upg := upgrader.New()
//	lis, _ := upg.Listen("tcp", ":8500")
//	srv := http.NewServer(http.WithListener(lis))
//
//	app := gofr.New(
//	    gofr.Upgrader(upg),
//	    gofr.Server(srv),
//	)
//	app.Run()
//
// On SIGHUP, the upgrader forks a child process that inherits the listening
// sockets. The parent waits for the child to signal readiness, then drains
// existing connections and exits.
package upgrader

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
)

const (
	envFDMeta  = "GOFR_FD_META"
	envReadyFD = "GOFR_READY_FD"
)

type meta struct {
	Network string `json:"n"`
	Address string `json:"a"`
}

// Upgrader manages listeners and coordinates parent/child handoff.
type Upgrader struct {
	isChild bool
	metas   []meta
	files   []*os.File // dup fds for inheritance
	seq     int        // position in metas for child recovery
	readyW  *os.File   // write end of ready pipe (child side)
}

// New creates an Upgrader. If the current process was started by a previous
// instance's Upgrade call, it recovers inherited listeners from file descriptors.
func New() *Upgrader {
	u := &Upgrader{}
	if raw := os.Getenv(envFDMeta); raw != "" {
		u.isChild = true
		json.Unmarshal([]byte(raw), &u.metas)
		if fdStr := os.Getenv(envReadyFD); fdStr != "" {
			var fd int
			fmt.Sscanf(fdStr, "%d", &fd)
			u.readyW = os.NewFile(uintptr(fd), "ready")
		}
	}
	return u
}

// IsChild returns true if this process was started via an Upgrade.
func (u *Upgrader) IsChild() bool {
	return u.isChild
}

// Listen creates or recovers a listener.
//
// In the parent, it calls net.Listen and stores a dup of the underlying
// file descriptor for later inheritance.
//
// In the child, it recovers the listener from the inherited fd.
func (u *Upgrader) Listen(network, addr string) (net.Listener, error) {
	if u.isChild {
		return u.recoverListener(network, addr)
	}
	return u.createListener(network, addr)
}

func (u *Upgrader) createListener(network, addr string) (net.Listener, error) {
	lis, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	// Get dup fd for inheritance. ExtraFiles[0]=readyPipe at fd3,
	// so first listener goes to fd4, second to fd5, etc.
	f, err := listenerFile(lis)
	if err != nil {
		lis.Close()
		return nil, err
	}
	u.files = append(u.files, f)
	u.metas = append(u.metas, meta{Network: network, Address: addr})
	return lis, nil
}

func (u *Upgrader) recoverListener(network, addr string) (net.Listener, error) {
	if u.seq >= len(u.metas) {
		return nil, fmt.Errorf("upgrader: no inherited fd for %s %s (seq=%d, total=%d)", network, addr, u.seq, len(u.metas))
	}
	// ExtraFiles layout: [0]=readyPipe(fd3), [1]=listener0(fd4), [2]=listener1(fd5)...
	fd := 4 + u.seq
	f := os.NewFile(uintptr(fd), "")
	defer f.Close()
	lis, err := net.FileListener(f)
	if err != nil {
		return nil, fmt.Errorf("upgrader: recover %s %s (fd=%d): %w", network, addr, fd, err)
	}
	u.seq++
	return lis, nil
}

// Ready signals the parent process that this child is fully started and
// ready to handle traffic. Only valid in the child process.
func (u *Upgrader) Ready() error {
	if u.readyW != nil {
		defer u.readyW.Close()
		_, err := u.readyW.Write([]byte("ready"))
		return err
	}
	return nil
}

// Upgrade forks a child process that inherits all registered listeners.
// It blocks until the child signals readiness, then returns.
// The parent should stop its servers and exit after Upgrade returns.
func (u *Upgrader) Upgrade() error {
	if len(u.files) == 0 {
		return fmt.Errorf("upgrader: no listeners to inherit")
	}

	// Create a pipe for ready notification.
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer r.Close()

	// ExtraFiles: [0]=ready pipe write end, [1..]=listener dup fds.
	extraFiles := append([]*os.File{w}, u.files...)

	metaRaw, _ := json.Marshal(u.metas)

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(),
		envFDMeta+"="+string(metaRaw),
		envReadyFD+"=3", // ExtraFiles[0] → fd 3
	)
	cmd.ExtraFiles = extraFiles
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Detach from parent process group so the child isn't killed
		// when the parent process group receives a signal.
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		w.Close()
		return err
	}
	w.Close() // parent doesn't write

	// Wait for child ready signal.
	buf := make([]byte, 16)
	n, err := r.Read(buf)
	if err != nil {
		return fmt.Errorf("upgrader: waiting for child ready: %w", err)
	}
	if string(buf[:n]) != "ready" {
		return fmt.Errorf("upgrader: unexpected child signal: %q", buf[:n])
	}

	// Close dup fds in parent — child has them now.
	for _, f := range u.files {
		f.Close()
	}
	u.files = nil

	return nil
}

// listenerFile returns a dup of the listener's underlying os.File.
func listenerFile(lis net.Listener) (*os.File, error) {
	switch l := lis.(type) {
	case *net.TCPListener:
		return l.File()
	default:
		return nil, fmt.Errorf("upgrader: unsupported listener type %T", lis)
	}
}
