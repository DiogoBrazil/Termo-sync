package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"termo-sync/internal/editorial"
)

type Usage struct {
	Prompt     int `json:"prompt_tokens"`
	Completion int `json:"completion_tokens"`
	Total      int `json:"total_tokens"`
}
type Revision struct {
	Model         string
	Usage         Usage
	Segments      []editorial.Segment
	Changes       []editorial.Change
	PromptVersion string
}

// Review sends one batch of the single deponent's transcript.
func (c *Client) Review(ctx context.Context, key, model string, input []editorial.Segment, authorized bool) (Revision, error) {
	var result Revision
	if !authorized {
		return result, errors.New("envio não autorizado")
	}
	if model != "deepseek-flash" && model != "deepseek-v4-pro" {
		return result, errors.New("modelo DeepSeek não permitido")
	}
	if len(input) == 0 {
		return result, errors.New("revisão sem segmentos")
	}
	data, err := json.Marshal(map[string]any{"segments": input})
	if err != nil {
		return result, err
	}
	instruction := editorial.Instruction()
	body, err := json.Marshal(map[string]any{"model": model, "thinking": map[string]string{"type": "disabled"}, "stream": false, "max_tokens": 16384, "response_format": map[string]string{"type": "json_object"}, "messages": []map[string]string{{"role": "system", "content": instruction}, {"role": "user", "content": string(data)}}})
	if err != nil {
		return result, err
	}
	var response struct {
		Model   string `json:"model"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *Usage `json:"usage"`
	}
	if err = c.request(ctx, "POST", "https://api.deepseek.com/chat/completions", key, bytes.NewReader(body), &response); err != nil {
		return result, err
	}
	if len(response.Choices) == 1 && response.Choices[0].Finish == "length" {
		return result, errors.New("o ajuste do texto foi interrompido por limite de tamanho; a proposta incompleta foi descartada")
	}
	if len(response.Choices) != 1 || response.Choices[0].Finish != "stop" {
		return result, errors.New("o ajuste do texto não foi concluído")
	}
	if response.Model == "" || response.Usage == nil || response.Usage.Prompt < 0 || response.Usage.Completion <= 0 || response.Usage.Total != response.Usage.Prompt+response.Usage.Completion {
		return result, errors.New("metadados de revisão ausentes ou inválidos")
	}
	choice := response.Choices[0]
	if choice.Message.Role != "assistant" || strings.TrimSpace(choice.Message.Content) == "" {
		return result, errors.New("revisão vazia ou inválida")
	}
	var output struct {
		Segments []editorial.Segment `json:"segments"`
	}
	if err = json.Unmarshal([]byte(choice.Message.Content), &output); err != nil {
		return result, errors.New("formato da proposta inválido")
	}
	changes, err := editorial.Compare(input, output.Segments)
	if err != nil {
		return result, err
	}
	return Revision{Model: response.Model, Usage: *response.Usage, Segments: output.Segments, Changes: changes, PromptVersion: editorial.Version}, nil
}
