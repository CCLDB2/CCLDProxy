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
// Devuelve la cabecera Upgrade y el User-Agent.
func readHeaders(br *bufio.Reader) (upgrade, agent string, err error) {
	for {
		line, e := readLine(br)
		if e != nil {
			return upgrade, agent, e
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
		}
	}
	return upgrade, agent, nil
}

// readAvailable lee lo que ya haya disponible en el buffer y del socket,
// con un breve timeout, sin bloquear indefinidamente.
func readAvailable(br *bufio.Reader, max int) []byte {
	var buf []byte
	if br.Buffered() > 0 {
		tmp := make([]byte, br.Buffered())
		if _, err := io.ReadFull(br, tmp); err == nil {
			buf = append(buf, tmp...)
		}
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
