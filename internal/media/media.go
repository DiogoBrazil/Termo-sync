package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type Track struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec_name"`
	Type     string `json:"codec_type"`
	Channels int    `json:"channels"`
	Rate     string `json:"sample_rate"`
	Tags     struct {
		Language string `json:"language"`
		Title    string `json:"title"`
	} `json:"tags"`
}
type Info struct {
	Streams []Track `json:"streams"`
	Format  struct {
		Name     string `json:"format_name"`
		Duration string `json:"duration"`
	} `json:"format"`
}
type Tools struct{ FFmpeg, FFprobe string }

// Locate uses only bundled tools beside the executable; it never executes a binary in the current directory.
func Locate() Tools {
	exe, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exe), "tools")
	return Tools{FFmpeg: filepath.Join(dir, "ffmpeg.exe"), FFprobe: filepath.Join(dir, "ffprobe.exe")}
}
func (t Tools) Inspect(ctx context.Context, path string) (Info, error) {
	var result Info
	stat, err := os.Stat(path)
	if err != nil || !stat.Mode().IsRegular() {
		return result, errors.New("arquivo de mídia inválido")
	}
	cmd := exec.CommandContext(ctx, t.FFprobe, "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mp3,mov,matroska,webm,wav,ogg,flac,aac", "-show_streams", "-show_format", "-of", "json", path)
	var out limitedBuffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		return result, errors.New("ffprobe indisponível ou mídia inválida")
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return result, errors.New("resposta ffprobe inválida")
	}
	audio := result.Streams[:0]
	for _, s := range result.Streams {
		if s.Type == "audio" {
			audio = append(audio, s)
		}
	}
	result.Streams = audio
	if len(audio) == 0 {
		return result, errors.New("arquivo sem áudio")
	}
	return result, nil
}

// Prepare validates decoding. A returned cleanup must always be deferred by the caller.
func (t Tools) Prepare(ctx context.Context, path string, info Info, index int) (string, func(), error) {
	noop := func() {}
	var track *Track
	for i := range info.Streams {
		if info.Streams[i].Index == index {
			track = &info.Streams[i]
		}
	}
	if track == nil {
		return "", noop, errors.New("trilha de áudio inexistente")
	}
	args := []string{"-nostdin", "-v", "error", "-xerror", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mp3,mov,matroska,webm,wav,ogg,flac,aac", "-i", path, "-map", fmt.Sprintf("0:%d", index), "-vn"}
	// MP3 containers with a single MP3 audio stream can be uploaded without rewriting.
	if info.Format.Name == "mp3" && track.Codec == "mp3" && len(info.Streams) == 1 {
		cmd := exec.CommandContext(ctx, t.FFmpeg, append(args, "-f", "null", "-")...)
		if err := cmd.Run(); err != nil {
			return "", noop, errors.New("falha na validação do áudio")
		}
		return path, noop, nil
	}
	root, err := tempRoot()
	if err != nil {
		return "", noop, err
	}
	dir, err := os.MkdirTemp(root, "session-")
	if err != nil {
		return "", noop, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := privateDirectory(dir); err != nil {
		cleanup()
		return "", noop, err
	}
	output := filepath.Join(dir, "audio.mp3")
	if track.Channels < 1 || track.Channels > 2 {
		cleanup()
		return "", noop, errors.New("MP3 suporta até dois canais; conversão automática recusada para preservar canais")
	}
	rate, err := strconv.Atoi(track.Rate)
	if err != nil {
		cleanup()
		return "", noop, errors.New("taxa de áudio inválida")
	}
	supported := map[int]bool{8000: true, 11025: true, 12000: true, 16000: true, 22050: true, 24000: true, 32000: true, 44100: true, 48000: true}
	if !supported[rate] {
		cleanup()
		return "", noop, errors.New("taxa incompatível com MP3; conversão automática recusada")
	}
	args = append(args, "-c:a", "libmp3lame", "-q:a", "2", "-ac", strconv.Itoa(track.Channels), "-ar", track.Rate, "-n", output)
	if err := exec.CommandContext(ctx, t.FFmpeg, args...).Run(); err != nil {
		cleanup()
		if ctx.Err() != nil {
			return "", noop, ctx.Err()
		}
		return "", noop, errors.New("falha na conversão para MP3")
	}
	return output, cleanup, nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4<<20 {
		return 0, errors.New("saída de inspeção excessiva")
	}
	return b.Buffer.Write(p)
}
