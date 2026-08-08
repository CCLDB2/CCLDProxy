package main

import (
	"bufio"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// readLine lee una linea hasta \n (sin el terminador).
func readLine(br *bufio.Reader) (string, error) {
	line, err := br.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// readHeaders lee headers HTTP hasta la linea en blanco.
// Devuelve Upgrade, User-Agent y Sec-WebSocket-Key.
func readHeaders(br *bufio.Reader) (upgrade, agent, wsKey string, err error) {
	for {
		line, e := readLine(br)
		if e != nil {
			return upgrade, agent, wsKey, e
		}
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "upgrade:"):
			upgrade = strings.TrimSpace(line[len("upgrade:"):])
		case strings.HasPrefix(lower, "user-agent:"):
			agent = strings.TrimSpace(line[len("user-agent:"):])
		case strings.HasPrefix(lower, "sec-websocket-key:"):
			wsKey = strings.TrimSpace(line[len("sec-websocket-key:"):])
		}
	}
	return upgrade, agent, wsKey, nil
}

// readAvailable espera el payload inicial del cliente (con timeout) para
// poder detectar el protocolo. Las apps envian GET+Upgrade, esperan el 101,
// y recien entonces mandan el payload. Por eso hay que esperar en el socket.
func readAvailable(br *bufio.Reader, conn net.Conn, timeout time.Duration, max int) []byte {
	var buf []byte

	// 1) Lo que ya quede en el buffer (si envio payload junto con el request)
	if br.Buffered() > 0 {
		tmp := make([]byte, br.Buffered())
		if _, err := io.ReadFull(br, tmp); err == nil {
			buf = append(buf, tmp...)
		}
	}
	// 2) Esperar mas datos en el socket con timeout
	if conn != nil && len(buf) == 0 {
		buf = make([]byte, 0)
		tmp := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(timeout))
		for {
			n, err := conn.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				if len(buf) >= 4 { // suficiente para detectar SSH/OpenVPN/V2Ray
					break
				}
			}
			if err != nil {
				break
			}
		}
		conn.SetReadDeadline(time.Time{})
	}
	if max > 0 && len(buf) > max {
		buf = buf[:max]
	}
	return buf
}

// bridge reenvia datos en ambos sentidos hasta que un lado se cierra.
func bridge(a, b net.Conn, bufSize int, logger *logger, ip string) {
	var wg sync.WaitGroup
	done := make(chan struct{}, 2)

	copyFn := func(dst, src net.Conn) {
		defer wg.Done()
		buf := make([]byte, bufSize)
		for {
			src.SetReadDeadline(time.Now().Add(2 * time.Minute))
			n, err := src.Read(buf)
			if n > 0 {
				if _, werr := dst.Write(buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
		dst.Close()
	}

	wg.Add(2)
	go copyFn(b, a)
	go copyFn(a, b)

	<-done
	a.Close()
	b.Close()
	wg.Wait()
}

func trim(s string) string {
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
