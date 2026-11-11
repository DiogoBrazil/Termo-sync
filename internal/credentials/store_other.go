//go:build !windows

package credentials

import "errors"

type Native struct{}

func (Native) Set(string, string) error   { return errors.New("Credential Manager requer Windows") }
func (Native) Get(string) (string, error) { return "", ErrMissing }
func (Native) Delete(string) error        { return errors.New("Credential Manager requer Windows") }
