package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
)

// fakeTransport devuelve los mensajes de incoming en orden y guarda los que manda el cliente. Sin mensajes,
// la conexión se cortó.
type fakeTransport struct {
	incoming []string
	sent     []map[string]any
}

func (f *fakeTransport) writeText(_ context.Context, msg []byte) error {
	var m map[string]any
	if err := json.Unmarshal(msg, &m); err != nil {
		return err
	}
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakeTransport) readMessage(_ context.Context) ([]byte, error) {
	if len(f.incoming) == 0 {
		return nil, io.EOF
	}
	m := f.incoming[0]
	f.incoming = f.incoming[1:]
	return []byte(m), nil
}

func (f *fakeTransport) close() error { return nil }

// connect manda el connect de DDP 1 y espera el connected, salteando el server_id que manda Meteor antes.
func TestDDPConnect(t *testing.T) {
	ft := &fakeTransport{incoming: []string{`{"server_id":"0"}`, `{"msg":"connected","session":"s1"}`}}

	if err := (&ddpClient{t: ft}).connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	want := []map[string]any{{"msg": "connect", "version": "1", "support": []any{"1"}}}
	if !reflect.DeepEqual(ft.sent, want) {
		t.Errorf("se mandó %v, se esperaba %v", ft.sent, want)
	}
}

// call manda el método con params como único argumento y devuelve el resultado de su id: contesta el ping,
// y saltea los mensajes de suscripciones y el resultado de otro método.
func TestDDPCall(t *testing.T) {
	ft := &fakeTransport{incoming: []string{
		`{"msg":"ping","id":"7"}`,
		`{"msg":"added","collection":"bondBound","id":"b1"}`,
		`{"msg":"result","id":"9","result":1}`,
		`{"msg":"updated","methods":["1"]}`,
		`{"msg":"result","id":"1","result":{"guides":[]}}`,
	}}
	c := &ddpClient{t: ft}

	got, err := c.call(context.Background(), "bondObject.getDataChart", map[string]any{"preview": true})

	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(got) != `{"guides":[]}` {
		t.Errorf("resultado %s, se esperaba {\"guides\":[]}", got)
	}
	want := []map[string]any{
		{"msg": "method", "method": "bondObject.getDataChart", "params": []any{map[string]any{"preview": true}}, "id": "1"},
		{"msg": "pong", "id": "7"},
	}
	if !reflect.DeepEqual(ft.sent, want) {
		t.Errorf("se mandó %v, se esperaba %v", ft.sent, want)
	}
}

// Si el método falla, call devuelve un *methodError con el código y el mensaje de Meteor; si se corta la
// conexión, el error no es un *methodError.
func TestDDPCallErrors(t *testing.T) {
	ft := &fakeTransport{incoming: []string{
		`{"msg":"result","id":"1","error":{"error":-15,"reason":"NO_SESSION","message":"NO_SESSION [-15]"}}`,
	}}
	c := &ddpClient{t: ft}

	_, err := c.call(context.Background(), "bondBound.preSale", map[string]any{})
	var me *methodError
	if !errors.As(err, &me) || me.Code != float64(-15) || me.Error() != "NO_SESSION [-15]" {
		t.Errorf("call = %v, se esperaba el error -15 de Meteor", err)
	}

	_, err = c.call(context.Background(), "bondBound.preSale", map[string]any{})
	if !errors.Is(err, io.EOF) || errors.As(err, &me) {
		t.Errorf("con la conexión cortada call = %v, se esperaba io.EOF", err)
	}
}
