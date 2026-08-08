package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
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

	// Leer headers hasta linea en blanco, capturando Upgrade, User-Agent y key
	_, agent, wsKey, rawHdr, err := readHeaders(br)
	if err != nil {
		return
	}
	c.agent = agent
	s.logger.Infof("headers: %s", trim(rawHdr))

	// Detectar si la conexion viene de Cloudflare (cf-ray / cdn-loop).
	// CF exige un handshake websocket RFC 6455 completo para abrir el tunel;
	// sin el, intercepta y no reenvia el payload. La conexion directa solo
	// necesita el 101 simple.
	viaCF := strings.Contains(rawHdr, "cloudflare") || strings.Contains(rawHdr, "cf-ray")

	// Responder handshake: completo RFC 6455 si viene de Cloudflare, 101 simple si directo.
	if err := writeUpgrade(c.conn, wsKey, viaCF); err != nil {
		return
	}

	// Detectar protocolo por el primer payload. Esperamos un timeout CORTO a
	// que el cliente envie el payload tras recibir el 101. Algunos clientes
	// (NPV a traves de Cloudflare) NO envian su payload hasta que reciben el
	// banner del servidor backend primero. Por eso, si no llega payload a
	// tiempo, asumimos SSH y dejamos que el bridge (el servidor OpenSSH habla
	// primero) desbloquee la conexion.
	payload := readAvailable(br, c.conn, 1200*time.Millisecond, 0, func(msg string) {
		s.logger.Infof("recv: %s", msg)
	})
	s.logger.Infof("payload(%d): %s", len(payload), hexDump(payload))
	backend := detectBackend(payload, s.cfg.Backend)
	if backend == nil {
		// Sin payload del cliente: el cliente esta esperando el banner del
		// servidor SSH. Conectamos al backend SSH y el bridge lo desbloquea.
		backend = findBackend(s.cfg.Backend, "SSH")
		if backend == nil {
			s.logger.Errorf("protocolo no reconocido de %s", c.ip)
			return
		}
		s.logger.Infof("sin payload del cliente; asumiendo SSH (espera banner del servidor)")
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

// findBackend devuelve el backend por nombre (ej: "SSH").
func findBackend(backends []Backend, name string) *Backend {
	for i := range backends {
		if backends[i].Name == name {
			return &backends[i]
		}
	}
	return nil
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
