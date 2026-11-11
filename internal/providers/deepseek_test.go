package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"termo-sync/internal/editorial"
	"testing"
)

func TestReviewValidation(t *testing.T) {
	input := []editorial.Segment{{ID: "s000000", Speaker: "B", Text: "Não vi 12 pessoas."}}
	for _, tc := range []struct {
		name, finish, content string
		usage                 bool
		fail                  bool
	}{
		{"complete", "stop", `{"segments":[{"id":"s000000","speaker":"B","text":"Não vi 12 pessoas."}]}`, true, false},
		{"truncated", "length", `{}`, true, true},
		{"missing usage", "stop", `{}`, false, true},
		{"missing segment", "stop", `{"segments":[]}`, true, true},
		{"wrong speaker", "stop", `{"segments":[{"id":"s000000","speaker":"A","text":"Não."}]}`, true, true},
		{"empty", "stop", " ", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New()
			calls := 0
			c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Authorization") != "Bearer key" {
					t.Fatal("bearer absent")
				}
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				if request["stream"] != false || request["thinking"].(map[string]any)["type"] != "disabled" {
					t.Fatal("wrong configuration")
				}
				response := map[string]any{"model": "deepseek-flash", "choices": []any{map[string]any{"finish_reason": tc.finish, "message": map[string]string{"role": "assistant", "content": tc.content}}}}
				if tc.usage {
					response["usage"] = map[string]int{"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}
				}
				data, _ := json.Marshal(response)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data))), Header: make(http.Header), Request: r}, nil
			})
			result, err := c.Review(context.Background(), "key", "deepseek-flash", input, true)
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected result %v", err)
			}
			if calls != 1 {
				t.Fatal("POST repeated")
			}
			if !tc.fail && (len(result.Segments) != 1 || result.PromptVersion == "") {
				t.Fatal("missing metadata")
			}
		})
	}
}
