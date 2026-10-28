package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// requestTimeout es lo que puede tardar cada llamada a un servicio.
const requestTimeout = 30 * time.Second

// maxBody es el tamaño máximo de una respuesta HTTP que se lee.
const maxBody = 16 << 20

// httpClient llama al API Go, con el x-secret del API y el token de una sesión, y a ms-feed, que no pide
// credenciales.
type httpClient struct {
	client  *http.Client
	goURL   string
	feedURL string
	secret  string
}

// httpResponse es el status y el cuerpo de una respuesta.
type httpResponse struct {
	status int
	body   []byte
}

// goGet hace GET a path del API Go con la sesión de token.
func (c *httpClient) goGet(ctx context.Context, path, token string) (httpResponse, error) {
	return c.get(ctx, c.goURL+path, map[string]string{"x-secret": c.secret, "x-token": token})
}

// feedGet hace GET a path de ms-feed.
func (c *httpClient) feedGet(ctx context.Context, path string) (httpResponse, error) {
	return c.get(ctx, c.feedURL+path, nil)
}

// get hace el GET. Un error es de la conexión; un status de error vuelve en la respuesta.
func (c *httpClient) get(ctx context.Context, url string, headers map[string]string) (httpResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return httpResponse{}, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return httpResponse{}, err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return httpResponse{}, fmt.Errorf("GET %s: %w", req.URL.Path, err)
	}
	return httpResponse{status: res.StatusCode, body: body}, nil
}

// apiErrorMessage devuelve el message de una respuesta de error del API Go, o el cuerpo si no lo tiene.
func apiErrorMessage(body []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return e.Message
	}
	return string(body)
}

// decodeJSON decodifica data en v y conserva los números como json.Number, para no perder decimales.
func decodeJSON(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	return d.Decode(v)
}
