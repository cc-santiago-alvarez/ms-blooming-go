package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// transport lleva los mensajes DDP: en la corrida es un websocket y en los tests, un fake.
type transport interface {
	writeText(ctx context.Context, msg []byte) error
	readMessage(ctx context.Context) ([]byte, error)
	close() error
}

// ddpClient llama métodos de Meteor por DDP, de a uno por vez.
type ddpClient struct {
	t      transport
	nextID int
}

// methodError es el error con que responde un método de Meteor.
type methodError struct {
	Code    any    `json:"error"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func (e *methodError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%v %s", e.Code, e.Reason)
}

// ddpMessage son los campos de un mensaje DDP que usa el cliente.
type ddpMessage struct {
	Msg    string          `json:"msg"`
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *methodError    `json:"error"`
}

// dialDDP abre el websocket de Meteor en rawURL y la sesión DDP.
func dialDDP(ctx context.Context, rawURL string) (*ddpClient, error) {
	ws, err := dialWS(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	c := &ddpClient{t: ws}
	if err := c.connect(ctx); err != nil {
		_ = ws.close()
		return nil, err
	}
	return c, nil
}

// connect abre la sesión DDP.
func (c *ddpClient) connect(ctx context.Context) error {
	if err := c.send(ctx, map[string]any{"msg": "connect", "version": "1", "support": []string{"1"}}); err != nil {
		return err
	}
	for {
		m, err := c.read(ctx)
		if err != nil {
			return err
		}
		switch m.Msg {
		case "connected":
			return nil
		case "failed":
			return errors.New("Meteor rechazó la conexión DDP")
		}
	}
}

// call ejecuta method con params como único argumento y devuelve su resultado. Si el método falla devuelve un
// *methodError; cualquier otro error es de la conexión.
func (c *ddpClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.nextID++
	id := strconv.Itoa(c.nextID)
	msg := map[string]any{"msg": "method", "method": method, "params": []any{params}, "id": id}
	if err := c.send(ctx, msg); err != nil {
		return nil, err
	}
	for {
		m, err := c.read(ctx)
		if err != nil {
			return nil, err
		}
		if m.Msg != "result" || m.ID != id {
			continue
		}
		if m.Error != nil {
			return nil, m.Error
		}
		return m.Result, nil
	}
}

// read devuelve el próximo mensaje del servidor y contesta sus ping.
func (c *ddpClient) read(ctx context.Context) (ddpMessage, error) {
	for {
		raw, err := c.t.readMessage(ctx)
		if err != nil {
			return ddpMessage{}, err
		}
		var m ddpMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return ddpMessage{}, fmt.Errorf("mensaje DDP inválido: %w", err)
		}
		if m.Msg != "ping" {
			return m, nil
		}
		pong := map[string]any{"msg": "pong"}
		if m.ID != "" {
			pong["id"] = m.ID
		}
		if err := c.send(ctx, pong); err != nil {
			return ddpMessage{}, err
		}
	}
}

// send manda msg como JSON.
func (c *ddpClient) send(ctx context.Context, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.t.writeText(ctx, data)
}

// close cierra la conexión con Meteor.
func (c *ddpClient) close() error {
	return c.t.close()
}
