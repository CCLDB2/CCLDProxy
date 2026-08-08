package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"time"
)

// Server maneja conexiones HTTP entrantes y las convierte en tuneles.
type Server struct {
	cfg     *Config
	logger  *logger
	limiter *limiter
	conns   map[int64]*Client
	idSeq   int64
}

// Client es una conexion de cliente aceptada.
type Client struct {
	id    int64
	conn  net.Conn
	addr  string
	ip    string
	agent string
}

func (s *Server) Serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			s.logger.Errorf("accept: %v", err)
			continue
		}
		ip := hostIP(conn.RemoteAddr())
		if !s.limiter.acquire(ip) {
			s.logger.Errorf("limite de conexiones por IP superado: %s", ip)
			conn.Close()
			continue
		}
		s.idSeq++
		c := &Client{id: s.idSeq, conn: conn, addr: conn.RemoteAddr().String(), ip: ip}
		s.conns[c.id] = c
		go s.handleClient(c)
	}
}

func (s *Server) handleClient(c *Client) {
	defer func() {
		s.limiter.release(c.ip)
		delete(s.conns, c.id)
		c.conn.Close()
	}()

	// Leer la linea de request HTTP
	br := bufio.NewReaderSize(c.conn, s.cfg.BufferSize)
	line, err := readLine(br)
	if err != nil {
		return
	}
	s.logger.Infof("cliente %s pidio: %s", c.ip, trim(line))

	// Leer headers hasta linea en blanco, capturando Upgrade y User-Agent
	_, agent, err := readHeaders(br)
	if err != nil {
		return
	}
	c.agent = agent

	// Siempre responder 101 (upgrade), como hace el proxy original.
	if _, err := io.WriteString(c.conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"); err != nil {
		return
	}

	// Detectar protocolo por el primer payload. Esperamos hasta 3s a que
	// el cliente envie el payload tras recibir el 101.
	payload := readAvailable(br, c.conn, 3*time.Second, 0)
	backend := detectBackend(payload, s.cfg.Backend)
	if backend == nil {
		s.logger.Errorf("protocolo no reconocido de %s", c.ip)
		return
	}
	s.logger.Infof("Modo: %s -> %s:%d", backend.Name, backend.Host, backend.Port)

	// Conectar al backend con timeout
	addr := fmt.Sprintf("%s:%d", backend.Host, backend.Port)
	up, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		s.logger.Errorf("Erro: Failed to connect to the server %s: %v", addr, err)
		return
	}
	defer up.Close()

	// Enviar el payload inicial capturado
	if len(payload) > 0 {
		up.Write(payload)
	}

	// Reenvio bidireccional
	bridge(c.conn, up, s.cfg.BufferSize, s.logger, c.ip)
}

// detectBackend elige el protocolo por el payload inicial.
func detectBackend(payload []byte, backends []Backend) *Backend {
	if len(payload) == 0 {
		return nil
	}
	for i := range backends {
		b := &backends[i]
		if b.isValid(payload) {
			return b
		}
	}
	return nil
}

// isValid implementa los detectores del proxy original.
func (b *Backend) isValid(payload []byte) bool {
	switch b.Name {
	case "SSH":
		// "SSH-" al inicio
		return len(payload) >= 4 && bytes.HasPrefix(payload, []byte("SSH-"))
	case "OpenVPN":
		// byte 0 = 0x00 y byte 1 = 0x38 ('8')  (formato OpenVPN)
		return len(payload) >= 2 && payload[0] == 0x00 && payload[1] == 0x38
	case "V2Ray":
		// byte 0 = 0x00 (formato V2Ray)
		return len(payload) >= 1 && payload[0] == 0x00 && payload[1] != 0x38
	}
	return false
}

func hostIP(addr net.Addr) string {
	if t, ok := addr.(*net.TCPAddr); ok {
		return t.IP.String()
	}
	host, _, _ := net.SplitHostPort(addr.String())
	return host
}
