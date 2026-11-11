//go:build windows

package credentials

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

type Native struct{}
type credential struct {
	Flags          uint32
	Type           uint32
	TargetName     *uint16
	Comment        *uint16
	LastWritten    syscall.Filetime
	BlobSize       uint32
	Blob           *byte
	Persist        uint32
	AttributeCount uint32
	Attributes     uintptr
	Alias          *uint16
	UserName       *uint16
}

var advapi = syscall.NewLazyDLL("advapi32.dll")
var writeCred = advapi.NewProc("CredWriteW")
var readCred = advapi.NewProc("CredReadW")
var deleteCred = advapi.NewProc("CredDeleteW")
var freeCred = advapi.NewProc("CredFree")

func (Native) Set(provider, key string) error {
	name, err := target(provider)
	if err != nil {
		return err
	}
	if len(key) == 0 || len(key) > 2560 {
		return errors.New("API Key vazia ou longa demais")
	}
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	blob := []byte(key)
	defer func() {
		for i := range blob {
			blob[i] = 0
		}
	}()
	c := credential{Type: 1, TargetName: p, BlobSize: uint32(len(blob)), Blob: &blob[0], Persist: 2}
	ok, _, _ := writeCred.Call(uintptr(unsafe.Pointer(&c)), 0)
	runtime.KeepAlive(blob)
	runtime.KeepAlive(p)
	if ok == 0 {
		return errors.New("não foi possível salvar no Credential Manager")
	}
	return nil
}
func (Native) Get(provider string) (string, error) {
	name, err := target(provider)
	if err != nil {
		return "", err
	}
	p, _ := syscall.UTF16PtrFromString(name)
	var c *credential
	ok, _, _ := readCred.Call(uintptr(unsafe.Pointer(p)), 1, 0, uintptr(unsafe.Pointer(&c)))
	runtime.KeepAlive(p)
	if ok == 0 {
		return "", ErrMissing
	}
	defer freeCred.Call(uintptr(unsafe.Pointer(c)))
	if c.BlobSize == 0 || c.BlobSize > 2560 {
		return "", ErrMissing
	}
	return string(unsafe.Slice(c.Blob, int(c.BlobSize))), nil
}
func (Native) Delete(provider string) error {
	name, err := target(provider)
	if err != nil {
		return err
	}
	p, _ := syscall.UTF16PtrFromString(name)
	ok, _, e := deleteCred.Call(uintptr(unsafe.Pointer(p)), 1, 0)
	runtime.KeepAlive(p)
	if ok == 0 && e != syscall.Errno(1168) {
		return errors.New("não foi possível remover a API Key")
	}
	return nil
}
