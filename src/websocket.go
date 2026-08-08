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

// writeUpgrade responde el handshake del websocket replicando el formato
// que usa SSHPLUS (HTTP/1.1 101 WebSocket), que es el que las apps tipo
// "NPV tunnel" y Cloudflare aceptan para abrir el tunel de bytes.
// Si el cliente envio Sec-WebSocket-Key, ademas incluimos el Accept RFC 6455.
func writeUpgrade(conn net.Conn, wsKey string) error {
	accept := ""
	if wsKey != "" {
		accept = "\r\nSec-WebSocket-Accept: " + wsAccept(wsKey)
	}
	resp := "HTTP/1.1 101 WebSocket\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade" +
		accept +
		"\r\n\r\n"
	_, err := conn.Write([]byte(resp))
	return err
}
