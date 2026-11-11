package domain

import "encoding/json"

type Utterance struct {
	Speaker string `json:"speaker"`
	Start   int64  `json:"start"`
	End     int64  `json:"end"`
	Text    string `json:"text"`
}

type Transcript struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	Text       string          `json:"text"`
	Error      string          `json:"error"`
	Model      string          `json:"speech_model_used"`
	Utterances []Utterance     `json:"utterances"`
	Raw        json.RawMessage `json:"-"`
}

func (t *Transcript) UnmarshalJSON(data []byte) error {
	type plain Transcript
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*t = Transcript(decoded)
	t.Raw = append(json.RawMessage(nil), data...)
	return nil
}
