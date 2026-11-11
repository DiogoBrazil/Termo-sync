// Package credentials keeps provider secrets out of presentation state.
package credentials

import "errors"

var ErrMissing = errors.New("API Key não configurada")

func target(provider string) (string, error) {
	switch provider {
	case "assemblyai", "deepseek":
		return "DepoimentoStudio/" + provider, nil
	}
	return "", errors.New("provedor desconhecido")
}

type Store interface {
	Set(string, string) error
	Get(string) (string, error)
	Delete(string) error
}
