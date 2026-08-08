package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
)

// wsMagic es el GUID fijo que exige RFC 6455 para el handshake.
const wsMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsAccept calcula el valor de Sec-WebSocket-Accept a partir de la key.
func wsAccept(key string) string {
	h := sha1.Sum([]byte(key + wsMagic))
	return base64.StdEncoding.EncodeToString(h[:])
}

// writeUpgrade responde el handshake del websocket. Si el cliente envio
// Sec-WebSocket-Key, devolvemos un upgrade RFC 6455 completo (necesario para
// que Cloudflare/cloudflared reconozcan el websocket y reenvien el payload).
// Si no hay key, respondemos un 101 generico (compatibilidad con clientes
// que no usan websocket estandar).
func writeUpgrade(conn net.Conn, wsKey string) error {
	if wsKey == "" {
		_, err := conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
		return err
	}
	accept := wsAccept(wsKey)
	resp := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"\r\n",
		accept,
	)
	_, err := conn.Write([]byte(resp))
	return err
}
