package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// TLSServer sirve tuneles sobre HTTPS (upgrade dentro de TLS).
type TLSServer struct {
	cfg    *Config
	logger *logger
}

func (s *TLSServer) Serve(ln net.Listener) {
	cert, err := tls.LoadX509KeyPair(s.cfg.Cert, s.cfg.Key)
	if err != nil {
		s.logger.Fatalf("HTTPS: no se pudo cargar cert/key (%s, %s): %v", s.cfg.Cert, s.cfg.Key, err)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	tlsLn := tls.NewListener(ln, tlsCfg)
	for {
		conn, err := tlsLn.Accept()
		if err != nil {
			s.logger.Errorf("accept tls: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *TLSServer) handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	br := bufio.NewReaderSize(conn, s.cfg.BufferSize)

	line, err := readLine(br)
	if err != nil {
		return
	}
	s.logger.Infof("TLS cliente: %s", trim(line))
	_, _, wsKey, err := readHeaders(br)
	if err != nil {
		return
	}
	// Limpiar deadline tras el handshake HTTP
	conn.SetDeadline(time.Time{})

	if err := writeUpgrade(conn, wsKey); err != nil {
		return
	}

	payload := readAvailable(br, conn, 3*time.Second, 0)
	backend := detectBackend(payload, s.cfg.Backend)
	if backend == nil {
		s.logger.Errorf("protocolo no reconocido")
		return
	}
	addr := fmt.Sprintf("%s:%d", backend.Host, backend.Port)
	up, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		s.logger.Errorf("Erro: Failed to connect to the server %s: %v", addr, err)
		return
	}
	defer up.Close()
	if len(payload) > 0 {
		up.Write(payload)
	}
	bridge(conn, up, s.cfg.BufferSize, s.logger, hostIP(conn.RemoteAddr()))
}

var _ = strings.TrimSpace
