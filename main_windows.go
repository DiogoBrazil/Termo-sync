//go:build windows

package main

import (
	"embed"
	"errors"
	"github.com/wailsapp/wails/v3/pkg/application"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	app "termo-sync/internal/application"
	"termo-sync/internal/credentials"
	"termo-sync/internal/media"
	"termo-sync/internal/providers"
)

//go:embed all:frontend/dist
var assets embed.FS

type Bridge struct {
	backend *app.Service
	desktop *application.App
	window  application.Window
}

func isCancelledDialog(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "cancel")
}

func (b *Bridge) Settings() app.Settings             { return b.backend.Settings() }
func (b *Bridge) SaveKey(provider, key string) error { return b.backend.SaveKey(provider, key) }
func (b *Bridge) RemoveKey(provider string) error    { return b.backend.RemoveKey(provider) }
func (b *Bridge) SetModel(model string) error        { return b.backend.SetModel(model) }
func (b *Bridge) View() app.View                     { return b.backend.View() }
func (b *Bridge) SelectFile() error {
	dlg := b.desktop.Dialog.OpenFile().SetTitle("Selecionar áudio ou vídeo").AddFilter("Áudio e vídeo", "*.mp3;*.wav;*.m4a;*.aac;*.ogg;*.opus;*.flac;*.wma;*.mp4;*.mkv;*.mov;*.webm;*.avi;*.wmv")
	if b.window != nil {
		dlg.AttachToWindow(b.window)
	}
	path, err := dlg.PromptForSingleSelection()
	if isCancelledDialog(err) || path == "" {
		return nil
	}
	if err != nil {
		return errors.New("não foi possível selecionar o arquivo")
	}
	return b.backend.Open(path)
}
func (b *Bridge) SelectTrack(id uint64, track int) error { return b.backend.SelectTrack(id, track) }
func (b *Bridge) Start(id uint64) error                  { return b.backend.Start(id) }
func (b *Bridge) RetryReview(id uint64) error            { return b.backend.RetryReview(id) }
func (b *Bridge) Cancel(id uint64) error                 { return b.backend.Cancel(id) }
func (b *Bridge) Edit(id uint64, text string) error      { return b.backend.Edit(id, text) }
func (b *Bridge) DeleteRemote(id uint64, confirmed bool) error {
	return b.backend.DeleteRemote(id, confirmed)
}
func (b *Bridge) KeepRemote(id uint64, confirmed bool) error {
	return b.backend.KeepRemote(id, confirmed)
}
func (b *Bridge) Copy() error {
	text, err := b.backend.Text()
	if err != nil {
		return err
	}
	if !b.desktop.Clipboard.SetText(text) {
		return errors.New("não foi possível copiar")
	}
	return nil
}
func (b *Bridge) CopyOriginal() error {
	text, err := b.backend.OriginalText()
	if err != nil {
		return err
	}
	if !b.desktop.Clipboard.SetText(text) {
		return errors.New("não foi possível copiar")
	}
	return nil
}
func (b *Bridge) Export() error {
	text, err := b.backend.Text()
	if err != nil {
		return err
	}
	dlg := b.desktop.Dialog.SaveFile().SetMessage("Exportar proposta em TXT").SetFilename("depoimento-revisado.txt").AddFilter("Texto UTF-8", "*.txt")
	if b.window != nil {
		dlg.AttachToWindow(b.window)
	}
	path, err := dlg.PromptForSingleSelection()
	if isCancelledDialog(err) || path == "" {
		return nil
	}
	if err != nil {
		return errors.New("não foi possível escolher o destino")
	}
	if !strings.EqualFold(filepath.Ext(path), ".txt") {
		path += ".txt"
	}
	original := b.backend.View().Document.File
	if strings.EqualFold(filepath.Clean(path), filepath.Clean(original)) {
		return errors.New("não é permitido sobrescrever o original")
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")), 0600); err != nil {
		return errors.New("não foi possível gravar o TXT")
	}
	return nil
}
func main() {
	var webviewData string
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		webviewData = filepath.Join(cache, "DepoimentoStudio", "webview2")
		_ = os.MkdirAll(webviewData, 0700)
		_ = os.Chdir(cache)
	}
	backend := app.NewService(credentials.Native{}, providers.New(), media.Locate())
	backend.RecoverTemporaries()
	bridge := &Bridge{backend: backend}
	root, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return
	}
	files := application.BundledAssetFileServer(root)
	var desktop *application.App
	desktop = application.New(application.Options{
		Name: "Depoimento Studio", Description: "Transcrição e revisão conservadora de depoimentos",
		Windows:  application.WindowsOptions{WebviewUserDataPath: webviewData},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Services: []application.Service{application.NewService(bridge)},
		Assets: application.AssetOptions{DisableLogging: true, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; frame-src 'none'; base-uri 'none'; form-action 'none'")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			files.ServeHTTP(w, r)
		})},
		OnShutdown: backend.Shutdown,
		ShouldQuit: func() bool {
			current := backend.View()
			if current.Document.ID == 0 {
				return true
			}
			var accepted atomic.Bool
			dialog := desktop.Dialog.Question().SetTitle("Encerrar a sessão?").SetMessage("O documento e as edições desta sessão não são salvos automaticamente. Exporte o TXT antes de sair. Trabalhos ativos serão cancelados; registros remotos não serão excluídos. Deseja encerrar?")
			dialog.AddButton("Continuar no aplicativo").SetAsDefault().SetAsCancel()
			dialog.AddButton("Encerrar sessão").OnClick(func() { accepted.Store(true) })
			dialog.Show()
			return accepted.Load()
		},
	})
	bridge.desktop = desktop
	bridge.window = desktop.Window.NewWithOptions(application.WebviewWindowOptions{Title: "Depoimento Studio", Width: 1366, Height: 900, MinWidth: 900, MinHeight: 650, URL: "/"})
	_ = desktop.Run()
}
