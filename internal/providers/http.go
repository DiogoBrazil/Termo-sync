package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct{ HTTP *http.Client }

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirecionamento bloqueado") }}}
}
func (c *Client) request(ctx context.Context, method, url, key string, body io.Reader, out any) error {
	if err := allowedDestination(url); err != nil {
		return err
	}
	if key == "" {
		return errors.New("credencial ausente")
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", key)
	if strings.HasPrefix(url, "https://api.deepseek.com/") {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	req.Header.Set("Content-Type", "application/json")
	if url == "https://api.assemblyai.com/v2/upload" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	if c.HTTP == nil {
		return errors.New("cliente HTTP ausente")
	}
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("redirecionamento bloqueado") }
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("falha de conexão com provedor")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("provedor retornou HTTP %d", resp.StatusCode)
	}
	const limit = 64 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return errors.New("falha ao ler resposta do provedor")
	}
	if len(data) > limit {
		return errors.New("resposta do provedor excede o limite")
	}
	if out == nil && len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if !json.Valid(data) || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("resposta JSON inválida do provedor")
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return errors.New("resposta JSON incompatível do provedor")
	}
	return nil
}

func allowedDestination(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Host != "api.assemblyai.com" && u.Host != "api.deepseek.com") || u.Fragment != "" || u.RawQuery != "" {
		return errors.New("destino HTTP não autorizado")
	}
	if u.Host == "api.deepseek.com" && u.Path != "/chat/completions" {
		return errors.New("endpoint HTTP não autorizado")
	}
	return nil
}
