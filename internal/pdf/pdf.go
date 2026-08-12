// Package pdf renders an HTML file to PDF using headless Chrome (CDP).
package pdf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Chrome is the path to the headless-shell/Chrome binary.
var Chrome = envOr("HEADLESS_SHELL", "/headless-shell/headless-shell")

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Options controls the PDF export.
type Options struct {
	Landscape bool
	Wait      time.Duration // time for JS rendering after load
	Port      int
}

// Render loads htmlPath (a local file) and writes the PDF to outPath.
func Render(htmlPath, outPath string, opt Options) error {
	if opt.Wait == 0 {
		opt.Wait = 3 * time.Second
	}
	if opt.Port == 0 {
		// A free port, so that concurrent renders do not collide.
		p, err := freePort()
		if err != nil {
			return err
		}
		opt.Port = p
	}
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	pageURL := "file://" + abs

	udd, err := os.MkdirTemp("", "cdp-pdf-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(udd)

	cmd := exec.Command(Chrome,
		"--no-sandbox", "--disable-dev-shm-usage", "--headless=old",
		"--disable-gpu", "--user-data-dir="+udd,
		"--remote-debugging-port="+strconv.Itoa(opt.Port),
		"--remote-allow-origins=*", "about:blank")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start chrome: %w", err)
	}
	defer func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Wait()
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", opt.Port)
	if err := waitFor(base + "/json/version"); err != nil {
		return err
	}

	req, _ := http.NewRequest("PUT", base+"/json/new?"+url.QueryEscape(pageURL), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("new tab: %w", err)
	}
	defer resp.Body.Close()
	var tgt struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tgt); err != nil {
		return err
	}

	ws, err := wsDial(tgt.WebSocketDebuggerURL, 10*time.Second)
	if err != nil {
		return err
	}
	defer ws.Close()
	ws.SetDeadline(time.Now().Add(2 * time.Minute))

	id := 0
	call := func(method string, params map[string]any) (map[string]json.RawMessage, error) {
		id++
		msg, _ := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
		if err := ws.WriteText(string(msg)); err != nil {
			return nil, err
		}
		for {
			data, err := ws.ReadMessage()
			if err != nil {
				return nil, err
			}
			var r struct {
				ID     int                        `json:"id"`
				Result map[string]json.RawMessage `json:"result"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(data, &r); err != nil {
				continue
			}
			if r.ID != id {
				continue
			}
			if r.Error != nil {
				return nil, fmt.Errorf("%s: %s", method, r.Error.Message)
			}
			return r.Result, nil
		}
	}

	if _, err := call("Page.enable", nil); err != nil {
		return err
	}
	if _, err := call("Page.navigate", map[string]any{"url": pageURL}); err != nil {
		return err
	}
	time.Sleep(opt.Wait)

	res, err := call("Page.printToPDF", map[string]any{
		"landscape": opt.Landscape, "printBackground": true,
		"marginTop": 0, "marginBottom": 0, "marginLeft": 0, "marginRight": 0,
		"preferCSSPageSize": true,
	})
	if err != nil {
		return err
	}
	var b64 string
	if err := json.Unmarshal(res["data"], &b64); err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, raw, 0o644)
}

// freePort asks the kernel for an unused TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitFor(u string) error {
	for i := 0; i < 100; i++ {
		resp, err := http.Get(u)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("chrome not responding at %s", u)
}
