package shared_exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	requestTimeout    = 60 * time.Second
	maxBufferBytes    = 256 << 20
	maxErrorBodyBytes = 64 << 10
)

// ExporterClient genera reportes en ms-exporter (POST /export/excel/report/{report}).
type ExporterClient struct {
	baseURL string
	secret  string
	client  *http.Client
}

var _ shared_ports.ExporterPort = (*ExporterClient)(nil)

func NewExporterClient(baseURL, secret string) *ExporterClient {
	return &ExporterClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

// Excel devuelve el archivo del reporte. Ante cualquier falla loguea el detalle, sin payload, token
// ni secret, y devuelve solo ErrExportFailed.
func (c *ExporterClient) Excel(ctx context.Context, report string, payload any) ([]byte, error) {
	file, err := c.excel(ctx, report, payload)
	if err != nil {
		log.Printf("[ExporterPort] no se pudo generar %s: %v", report, err)
		return nil, shared_domain.ErrExportFailed
	}
	return file, nil
}

func (c *ExporterClient) excel(ctx context.Context, report string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := c.baseURL + "/export/excel/report/" + url.PathEscape(report)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-secret", c.secret)
	if session, ok := session_domain.GetUserSession(ctx); ok && session.Token != "" {
		req.Header.Set("x-token", session.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Se lee el body para que la conexión se pueda reusar.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("ms-exporter respondió %d", resp.StatusCode)
	}

	// buffer.toJSON() de Node: los bytes del archivo como arreglo de números.
	var buffer struct {
		Type string `json:"type"`
		Data []byte `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBufferBytes)).Decode(&buffer); err != nil {
		return nil, fmt.Errorf("respuesta inválida: %w", err)
	}
	if buffer.Type != "Buffer" {
		return nil, errors.New("la respuesta no es un Buffer")
	}
	return buffer.Data, nil
}
