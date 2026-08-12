package pdf

// Minimaler WebSocket-Client (RFC 6455), gerade genug fuer das
// Chrome-DevTools-Protokoll: Textframes, kein Deflate, kein Ping-Handling
// ausser Pong-Antworten.

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type wsConn struct {
	c  net.Conn
	br *bufio.Reader
}

func wsDial(rawURL string, timeout time.Duration) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	c, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, err
	}
	key := make([]byte, 16)
	rand.Read(key)
	k := base64.StdEncoding.EncodeToString(key)
	path := u.RequestURI()
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\n"+
		"Connection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		path, u.Host, k)
	if _, err := io.WriteString(c, req); err != nil {
		c.Close()
		return nil, err
	}
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		c.Close()
		return nil, fmt.Errorf("websocket: unerwarteter Status %s", resp.Status)
	}
	sum := sha1.Sum([]byte(k + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if resp.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(sum[:]) {
		c.Close()
		return nil, fmt.Errorf("websocket: falscher Accept-Header")
	}
	return &wsConn{c: c, br: br}, nil
}

func (w *wsConn) Close() error { return w.c.Close() }

func (w *wsConn) SetDeadline(t time.Time) { w.c.SetDeadline(t) }

func (w *wsConn) write(opcode byte, payload []byte) error {
	var h []byte
	n := len(payload)
	h = append(h, 0x80|opcode)
	switch {
	case n < 126:
		h = append(h, byte(0x80|n))
	case n < 1<<16:
		h = append(h, 0x80|126, byte(n>>8), byte(n))
	default:
		h = append(h, 0x80|127)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(n))
		h = append(h, b[:]...)
	}
	var mask [4]byte
	rand.Read(mask[:])
	h = append(h, mask[:]...)
	buf := make([]byte, 0, len(h)+n)
	buf = append(buf, h...)
	for i := 0; i < n; i++ {
		buf = append(buf, payload[i]^mask[i%4])
	}
	_, err := w.c.Write(buf)
	return err
}

// WriteText sendet einen Textframe.
func (w *wsConn) WriteText(s string) error { return w.write(1, []byte(s)) }

// ReadMessage liefert die naechste (ggf. defragmentierte) Text-/Binaernachricht.
func (w *wsConn) ReadMessage() ([]byte, error) {
	var msg []byte
	for {
		var hdr [2]byte
		if _, err := io.ReadFull(w.br, hdr[:]); err != nil {
			return nil, err
		}
		fin := hdr[0]&0x80 != 0
		opcode := hdr[0] & 0x0f
		n := int(hdr[1] & 0x7f)
		switch n {
		case 126:
			var b [2]byte
			if _, err := io.ReadFull(w.br, b[:]); err != nil {
				return nil, err
			}
			n = int(binary.BigEndian.Uint16(b[:]))
		case 127:
			var b [8]byte
			if _, err := io.ReadFull(w.br, b[:]); err != nil {
				return nil, err
			}
			n = int(binary.BigEndian.Uint64(b[:]))
		}
		payload := make([]byte, n)
		if _, err := io.ReadFull(w.br, payload); err != nil {
			return nil, err
		}
		switch opcode {
		case 8: // close
			return nil, io.EOF
		case 9: // ping
			if err := w.write(10, payload); err != nil {
				return nil, err
			}
			continue
		case 10: // pong
			continue
		}
		msg = append(msg, payload...)
		if fin {
			return msg, nil
		}
	}
}
