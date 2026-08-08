package main

import (
	"net"
)

// writeUpgrade responde el handshake replicando EXACTAMENTE la respuesta de
// SSHPLUS: solo "HTTP/1.1 101 WebSocket\r\n\r\n" sin headers de upgrade.
// La captura de SSHPLUS mostro una respuesta de 26 bytes (22 + \r\n\r\n).
// Cloudflare, al ver un 101 sin Upgrade/Connection, lo trata como tunel TCP
// crudo y pasa los bytes directamente (el payload fluye). Anadir headers
// Upgrade/Connection hace que Cloudflare intente validar el websocket y
// no entregue el payload.
func writeUpgrade(conn net.Conn, wsKey string) error {
	_, err := conn.Write([]byte("HTTP/1.1 101 WebSocket\r\n\r\n"))
	return err
}
