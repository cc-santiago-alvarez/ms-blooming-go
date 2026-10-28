package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

// Un frame armado con appendFrame se lee igual con readFrame, con y sin máscara, en los tres formatos de
// largo: hasta 125 bytes, con 2 bytes extra y con 8.
func TestFrameRoundTrip(t *testing.T) {
	tests := []struct {
		size       int
		headerSize int
	}{
		{size: 0, headerSize: 2},
		{size: 125, headerSize: 2},
		{size: 126, headerSize: 4},
		{size: 65535, headerSize: 4},
		{size: 65536, headerSize: 10},
	}
	for _, tt := range tests {
		for _, masked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d bytes con máscara %v", tt.size, masked), func(t *testing.T) {
				payload := make([]byte, tt.size)
				for i := range payload {
					payload[i] = byte(i % 251)
				}
				var mask []byte
				wantLen := tt.headerSize + tt.size
				if masked {
					mask = []byte{0x37, 0xfa, 0x21, 0x3d}
					wantLen += 4
				}

				data := appendFrame(nil, opText, payload, mask)
				if len(data) != wantLen {
					t.Errorf("el frame mide %d bytes, se esperaban %d", len(data), wantLen)
				}
				f, err := readFrame(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("readFrame: %v", err)
				}
				if !f.fin || f.opcode != opText || !bytes.Equal(f.payload, payload) {
					t.Errorf("se leyó fin=%v opcode=%d con %d bytes, se esperaba el frame final de texto original", f.fin, f.opcode, len(f.payload))
				}
			})
		}
	}
}

// El frame enmascarado de "Hello" es el del ejemplo de RFC 6455.
func TestAppendFrameMatchesRFCExample(t *testing.T) {
	got := appendFrame(nil, opText, []byte("Hello"), []byte{0x37, 0xfa, 0x21, 0x3d})
	want := []byte{0x81, 0x85, 0x37, 0xfa, 0x21, 0x3d, 0x7f, 0x9f, 0x4d, 0x51, 0x58}
	if !bytes.Equal(got, want) {
		t.Errorf("frame % x, se esperaba % x", got, want)
	}
}

// El Sec-WebSocket-Accept de la clave del ejemplo de RFC 6455.
func TestAcceptKey(t *testing.T) {
	if got := acceptKey("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Errorf("acceptKey = %q, se esperaba s3pPLMBiTxaQ9kYGzzhZRbK+xOo=", got)
	}
}

// readMessage une un mensaje fragmentado, contesta con un pong enmascarado el ping que llega en el medio y,
// cuando el servidor cierra, devuelve errWSClosed.
func TestReadMessage(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	c := &wsConn{conn: client, r: bufio.NewReader(client)}

	pong := make(chan frame, 1)
	go func() {
		var out []byte
		out = append(out, opText, 3)
		out = append(out, "Hel"...)
		out = appendFrame(out, opPing, []byte("p"), nil)
		out = append(out, 0x80|opContinuation, 2)
		out = append(out, "lo"...)
		if _, err := server.Write(out); err != nil {
			t.Errorf("el servidor no pudo escribir: %v", err)
			return
		}

		head := make([]byte, 2)
		if _, err := io.ReadFull(server, head); err != nil {
			t.Errorf("el servidor no recibió el pong: %v", err)
			return
		}
		if head[1]&0x80 == 0 {
			t.Error("el pong del cliente no está enmascarado")
		}
		f, err := readFrame(io.MultiReader(bytes.NewReader(head), server))
		if err != nil {
			t.Errorf("el servidor no pudo leer el pong: %v", err)
			return
		}
		pong <- f
		_, _ = server.Write(appendFrame(nil, opClose, nil, nil))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	msg, err := c.readMessage(ctx)
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if string(msg) != "Hello" {
		t.Errorf("mensaje %q, se esperaba \"Hello\"", msg)
	}
	select {
	case f := <-pong:
		if f.opcode != opPong || string(f.payload) != "p" {
			t.Errorf("el servidor recibió opcode %d con %q, se esperaba un pong con \"p\"", f.opcode, f.payload)
		}
	case <-ctx.Done():
		t.Fatal("el servidor no recibió el pong")
	}

	if _, err := c.readMessage(ctx); !errors.Is(err, errWSClosed) {
		t.Errorf("después del cierre readMessage = %v, se esperaba errWSClosed", err)
	}
}
