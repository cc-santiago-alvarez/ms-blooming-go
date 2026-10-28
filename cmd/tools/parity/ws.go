package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Opcodes de websocket que maneja el cliente.
const (
	opContinuation byte = 0x0
	opText         byte = 0x1
	opBinary       byte = 0x2
	opClose        byte = 0x8
	opPing         byte = 0x9
	opPong         byte = 0xA
)

// maxPayload es el tamaño máximo de un mensaje que acepta el cliente.
const maxPayload = 64 << 20

// wsGUID es el GUID con el que el servidor calcula Sec-WebSocket-Accept.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var errWSClosed = errors.New("el servidor cerró el websocket")

// wsConn es una conexión websocket de cliente, sin extensiones: manda mensajes de texto enmascarados y lee
// mensajes completos.
type wsConn struct {
	conn net.Conn
	r    *bufio.Reader
	mu   sync.Mutex
}

// dialWS abre la conexión a rawURL, que tiene que ser ws://, y hace el handshake.
func dialWS(ctx context.Context, rawURL string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ws" {
		return nil, fmt.Errorf("solo se admite ws://, llegó %q", rawURL)
	}
	host := u.Host
	if u.Port() == "" {
		host = net.JoinHostPort(u.Hostname(), "80")
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, err
	}
	ws, err := handshake(ctx, conn, u)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ws, nil
}

// handshake pide el cambio a websocket y verifica la respuesta del servidor.
func handshake(ctx context.Context, conn net.Conn, u *url.URL) (*wsConn, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(nonce)

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	req := "GET " + u.RequestURI() + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, err
	}

	r := bufio.NewReader(conn)
	res, err := http.ReadResponse(r, nil)
	if err != nil {
		return nil, fmt.Errorf("handshake websocket: %w", err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("handshake websocket: status %d", res.StatusCode)
	}
	if res.Header.Get("Sec-WebSocket-Accept") != acceptKey(key) {
		return nil, errors.New("handshake websocket: Sec-WebSocket-Accept inválido")
	}
	_ = conn.SetDeadline(time.Time{})
	return &wsConn{conn: conn, r: r}, nil
}

// acceptKey es el Sec-WebSocket-Accept que corresponde a key.
func acceptKey(key string) string {
	sum := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// appendFrame agrega a dst un frame final con opcode y payload. Con mask (4 bytes) lo enmascara, como tiene
// que mandarlo un cliente.
func appendFrame(dst []byte, opcode byte, payload, mask []byte) []byte {
	dst = append(dst, 0x80|opcode)
	var maskBit byte
	if mask != nil {
		maskBit = 0x80
	}
	switch n := len(payload); {
	case n <= 125:
		dst = append(dst, maskBit|byte(n))
	case n <= 0xFFFF:
		dst = append(dst, maskBit|126)
		dst = binary.BigEndian.AppendUint16(dst, uint16(n))
	default:
		dst = append(dst, maskBit|127)
		dst = binary.BigEndian.AppendUint64(dst, uint64(n))
	}
	if mask == nil {
		return append(dst, payload...)
	}
	dst = append(dst, mask...)
	start := len(dst)
	dst = append(dst, payload...)
	for i := range payload {
		dst[start+i] ^= mask[i%4]
	}
	return dst
}

// frame es un frame leído, con el payload ya desenmascarado.
type frame struct {
	fin     bool
	opcode  byte
	payload []byte
}

// readFrame lee un frame de r.
func readFrame(r io.Reader) (frame, error) {
	var head [2]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return frame{}, err
	}
	f := frame{fin: head[0]&0x80 != 0, opcode: head[0] & 0x0F}
	masked := head[1]&0x80 != 0

	n := uint64(head[1] & 0x7F)
	switch n {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return frame{}, err
		}
		n = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return frame{}, err
		}
		n = binary.BigEndian.Uint64(ext[:])
	}
	if n > maxPayload {
		return frame{}, fmt.Errorf("websocket: frame de %d bytes, supera el máximo", n)
	}

	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(r, mask[:]); err != nil {
			return frame{}, err
		}
	}
	f.payload = make([]byte, n)
	if _, err := io.ReadFull(r, f.payload); err != nil {
		return frame{}, err
	}
	if masked {
		for i := range f.payload {
			f.payload[i] ^= mask[i%4]
		}
	}
	return f, nil
}

// writeFrame manda un frame enmascarado con una máscara nueva.
func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.conn.Write(appendFrame(nil, opcode, payload, mask))
	return err
}

// writeText manda msg como un mensaje de texto.
func (c *wsConn) writeText(ctx context.Context, msg []byte) error {
	defer c.watch(ctx)()
	return c.writeFrame(opText, msg)
}

// readMessage devuelve el próximo mensaje completo, uniendo sus fragmentos. Contesta los ping con pong y falla
// si el servidor cierra la conexión.
func (c *wsConn) readMessage(ctx context.Context) ([]byte, error) {
	defer c.watch(ctx)()
	var msg []byte
	started := false
	for {
		f, err := readFrame(c.r)
		if err != nil {
			return nil, err
		}
		switch f.opcode {
		case opPing:
			if err := c.writeFrame(opPong, f.payload); err != nil {
				return nil, err
			}
			continue
		case opPong:
			continue
		case opClose:
			return nil, errWSClosed
		case opText, opBinary:
			if started {
				return nil, errors.New("websocket: mensaje nuevo antes de terminar el anterior")
			}
			started = true
			msg = f.payload
		case opContinuation:
			if !started {
				return nil, errors.New("websocket: continuación sin un mensaje empezado")
			}
			msg = append(msg, f.payload...)
		default:
			return nil, fmt.Errorf("websocket: opcode %d desconocido", f.opcode)
		}
		if len(msg) > maxPayload {
			return nil, fmt.Errorf("websocket: mensaje de %d bytes, supera el máximo", len(msg))
		}
		if f.fin {
			return msg, nil
		}
	}
}

// watch aplica el deadline de ctx a la conexión y la corta si ctx se cancela; devuelve la función que deja de
// vigilarlo.
func (c *wsConn) watch(ctx context.Context) func() {
	deadline, _ := ctx.Deadline()
	_ = c.conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { _ = c.conn.SetDeadline(time.Now()) })
	return func() { stop() }
}

// close avisa al servidor y cierra la conexión.
func (c *wsConn) close() error {
	_ = c.writeFrame(opClose, nil)
	return c.conn.Close()
}
