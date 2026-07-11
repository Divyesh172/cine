package resolver

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// Peerflix streams a magnet/torrent through a local peerflix HTTP server, which
// mpv can then play like any http:// URL.
//
// Install the bridge once:  npm install -g peerflix
type Peerflix struct {
	Bin     string        // binary name/path; defaults to "peerflix"
	Extra   []string      // extra CLI args from config
	Timeout time.Duration // how long to wait for the server to come up
}

// NewPeerflix builds a Peerflix resolver with sane defaults.
func NewPeerflix(bin string, extra []string, timeout time.Duration) *Peerflix {
	if bin == "" {
		bin = "peerflix"
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &Peerflix{Bin: bin, Extra: extra, Timeout: timeout}
}

// Resolve starts peerflix on a free port and returns its local stream URL plus a
// cleanup func that stops the process.
func (p *Peerflix) Resolve(magnet string) (string, func(), error) {
	port, err := freePort()
	if err != nil {
		return "", nil, err
	}
	// peerflix serves the largest media file in the torrent at http://host:port/.
	args := append([]string{magnet, "--port", strconv.Itoa(port), "--no-quit"}, p.Extra...)
	cmd := exec.Command(p.Bin, args...)
	cmd.Stdout = os.Stderr // peerflix progress -> our stderr, keep stdout clean
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("start %q: %w (install it once with: npm install -g peerflix)", p.Bin, err)
	}
	cleanup := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}
	url := fmt.Sprintf("http://localhost:%d/", port)
	if err := waitForServer(url, p.Timeout); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("peerflix stream did not come up within %s: %w", p.Timeout, err)
	}
	return url, cleanup, nil
}

// freePort asks the OS for an available TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForServer polls url until peerflix serves the file (or the deadline
// passes). A tiny Range request avoids pulling the whole file just to check.
func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Range", "bytes=0-1")
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out after %s", timeout)
}
