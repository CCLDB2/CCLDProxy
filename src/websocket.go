package main

import (
	"crypto/sha1"
	"encoding/base64"
	"net"
)

// wsMagic es el GUID fijo que exige RFC 6455 para el handshake.
const wsMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsAccept calcula el valor de Sec-WebSocket-Accept a partir de la key.
func wsAccept(key string) string {
	h := sha1.Sum([]byte(key + wsMagic))
	return base64.StdEncoding.EncodeToString(h[:])
}

// writeUpgrade responde el handshake del websocket. Hay dos casos:
//   - viaCF=false (conexion directa al VPS): basta el 101 simple tal como
//     responde SSHPLUS ("HTTP/1.1 101 WebSocket\r\n\r\n"). Es el que las
//     apps tipo NPV/HTTPInjector y el tunel directo aceptan.
//   - viaCF=true (detras de Cloudflare): CF intercepta el websocket y solo
//     abre el tunel si el origin completa el handshake RFC 6455 con
//     Sec-WebSocket-Accept. Sin el, CF no reenvia el payload al origin.
func writeUpgrade(conn net.Conn, wsKey string, viaCF bool) error {
	if !viaCF {
		_, err := conn.Write([]byte("HTTP/1.1 101 WebSocket\r\n\r\n"))
		return err
	}
	// Cloudflare: handshake RFC 6455 completo.
	key := wsKey
	if key == "" {
		key = "dGhlIHNhbXBsZSBub25jZQ==" // nonce por defecto si CF no la manda
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + wsAccept(key) + "\r\n\r\n"
	_, err := conn.Write([]byte(resp))
	return err
}
