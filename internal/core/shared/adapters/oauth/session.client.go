package oauth_adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

const (
	requestTimeout = 3 * time.Second
	clientTimeout  = 4 * time.Second
	maxBodyBytes   = 1 << 20
)

// SessionClient consulta la sesión en ms-oauth (GET /api/v1/session).
type SessionClient struct {
	url    string
	secret string
	client *http.Client
}

func NewSessionClient(url, secret string) *SessionClient {
	return &SessionClient{
		url:    url,
		secret: secret,
		client: &http.Client{Timeout: clientTimeout},
	}
}

// Verify devuelve la sesión del token. Cualquier respuesta distinta de 200 es un error; los errores
// no incluyen el token.
func (c *SessionClient) Verify(ctx context.Context, token string) (*session_domain.UserSession, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-token", token)
	req.Header.Set("x-secret", c.secret)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ms-oauth respondió %d", resp.StatusCode)
	}

	var session session_domain.UserSession
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&session); err != nil {
		return nil, fmt.Errorf("no se pudo decodificar la sesión: %w", err)
	}
	if session.UserID == "" {
		return nil, errors.New("la sesión no trae userId")
	}

	return &session, nil
}
