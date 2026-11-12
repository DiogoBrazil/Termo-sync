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
	"strconv"
	"strings"
	"time"
)

// StatusError carries only the HTTP status, a parsed Retry-After and the provider's own short
// diagnostic for rejected requests. It never includes credentials or transcript content.
type StatusError struct {
	Status     int
	RetryAfter time.Duration
	Detail     string
}

func (e *StatusError) Error() string {
	var text string
	switch {
	case e.Status == 401 || e.Status == 403:
		text = "API Key recusada pelo provedor. Confira a chave em Configurações"
	case e.Status == 402:
		text = "saldo insuficiente na conta do provedor"
	case e.Status == 404:
		text = "registro não encontrado no provedor"
	case e.Status == 413:
		text = "arquivo ou texto excede o limite aceito pelo provedor"
	case e.Status == 429:
		text = "limite de requisições do provedor atingido. Aguarde e tente novamente"
	case e.Status >= 500:
		text = "provedor temporariamente indisponível"
	default:
		text = "requisição recusada pelo provedor"
	}
	text = fmt.Sprintf("%s (HTTP %d)", text, e.Status)
	if e.Detail != "" {
		text += ": " + e.Detail
	}
	return text
}

// Transient reports whether repeating an idempotent request may succeed.
func (e *StatusError) Transient() bool { return e.Status == 429 || e.Status >= 500 }

var errConnection = errors.New("falha de conexão com o provedor. Confira a internet")

// detail extracts the provider's request diagnostic ("error" string or "error.message") from 4xx bodies.
func detail(data []byte) string {
	var body struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &body) != nil || len(body.Error) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(body.Error, &text) != nil {
		var nested struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body.Error, &nested) != nil {
			return ""
		}
		text = nested.Message
	}
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(text); len(r) > 300 {
		text = string(r[:300]) + "…"
	}
	return text
}

type Client struct {
	HTTP *http.Client
	// Poll overrides the transcript status interval (tests); zero means three seconds.
	Poll time.Duration
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirecionamento bloqueado") }}}
}
func (c *Client) request(ctx context.Context, method, url, key string, body io.Reader, out any) error {
	return c.send(ctx, method, url, key, body, -1, out)
}

// send performs one request. length >= 0 declares the body size so uploads are not chunked.
func (c *Client) send(ctx context.Context, method, url, key string, body io.Reader, length int64, out any) error {
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
	if length >= 0 {
		req.ContentLength = length
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
		return errConnection
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		failure := &StatusError{Status: resp.StatusCode}
		if seconds, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); err == nil && seconds > 0 {
			failure.RetryAfter = min(time.Duration(seconds)*time.Second, 2*time.Minute)
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 401 && resp.StatusCode != 403 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			failure.Detail = detail(data)
		}
		return failure
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
