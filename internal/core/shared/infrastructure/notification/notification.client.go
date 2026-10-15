package shared_notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	requestTimeout = 15 * time.Second
	maxBodyBytes   = 64 << 10
)

// NotificationClient envía notificaciones a ms-notification (POST /notify/{group}/{notification}).
type NotificationClient struct {
	baseURL string
	secret  string
	client  *http.Client
}

var _ shared_ports.NotificationPort = (*NotificationClient)(nil)

func NewNotificationClient(baseURL, secret string) *NotificationClient {
	return &NotificationClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

// Send envía la notificación aunque ctx ya esté cancelado. Un fallo solo se loguea, sin params,
// token ni secret.
func (c *NotificationClient) Send(ctx context.Context, group, notification string, params map[string]any) {
	if err := c.send(ctx, group, notification, params); err != nil {
		log.Printf("[NotificationPort] no se pudo enviar %s/%s: %v", group, notification, err)
	}
}

func (c *NotificationClient) send(ctx context.Context, group, notification string, params map[string]any) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), requestTimeout)
	defer cancel()

	if params == nil {
		params = map[string]any{}
	}
	body, err := json.Marshal(map[string]any{"data": map[string]any{"params": params}})
	if err != nil {
		return err
	}

	endpoint := c.baseURL + "/notify/" + url.PathEscape(group) + "/" + url.PathEscape(notification)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-secret", c.secret)
	if session, ok := session_domain.GetUserSession(ctx); ok && session.Token != "" {
		req.Header.Set("x-token", session.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Se lee el body para que la conexión se pueda reusar.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("ms-notification respondió %d", resp.StatusCode)
	}
	return nil
}
