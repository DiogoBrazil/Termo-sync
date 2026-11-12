package media

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// formats delimits the demuxers FFmpeg may use: MP3, MP4/MOV/M4A, MKV/WebM, WAV, OGG/Opus, FLAC, AAC, AVI and ASF (WMV/WMA).
const formats = "mp3,mov,matroska,webm,wav,ogg,flac,aac,avi,asf"

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
		Size     string `json:"size"`
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
	cmd := exec.CommandContext(ctx, t.FFprobe, "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", formats, "-show_streams", "-show_format", "-of", "json", path)
	var out limitedBuffer
	cmd.Stdout = &out
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		return result, errors.New("não foi possível ler a mídia: arquivo corrompido, formato sem suporte ou ffprobe ausente na pasta tools")
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

// run executes FFmpeg reporting real progress (processed time over total duration) when measurable.
func (t Tools) run(ctx context.Context, args []string, info Info, report func(float64)) error {
	total, _ := strconv.ParseFloat(info.Format.Duration, 64)
	cmd := exec.CommandContext(ctx, t.FFmpeg, append([]string{"-progress", "pipe:1", "-nostats"}, args...)...)
	hideWindow(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		value, found := strings.CutPrefix(scanner.Text(), "out_time_us=")
		if !found || report == nil || total <= 0 {
			continue
		}
		if us, err := strconv.ParseFloat(value, 64); err == nil && us >= 0 {
			report(min(us/1e6/total, 1))
		}
	}
	return cmd.Wait()
}

// Prepare validates decoding. A returned cleanup must always be deferred by the caller.
// report receives the stage ("validating" or "converting") and a fraction in [0,1].
func (t Tools) Prepare(ctx context.Context, path string, info Info, index int, report func(stage string, fraction float64)) (string, func(), error) {
	noop := func() {}
	if report == nil {
		report = func(string, float64) {}
	}
	var track *Track
	for i := range info.Streams {
		if info.Streams[i].Index == index {
			track = &info.Streams[i]
		}
	}
	if track == nil {
		return "", noop, errors.New("trilha de áudio inexistente")
	}
	args := []string{"-nostdin", "-v", "error", "-xerror", "-protocol_whitelist", "file,pipe", "-format_whitelist", formats, "-i", path, "-map", fmt.Sprintf("0:%d", index), "-vn"}
	// MP3 containers with a single MP3 audio stream can be uploaded without rewriting.
	if info.Format.Name == "mp3" && track.Codec == "mp3" && len(info.Streams) == 1 {
		report("validating", 0)
		if err := t.run(ctx, append(args, "-f", "null", "-"), info, func(f float64) { report("validating", f) }); err != nil {
			if ctx.Err() != nil {
				return "", noop, ctx.Err()
			}
			return "", noop, errors.New("falha na validação do áudio: o MP3 parece corrompido")
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
	report("converting", 0)
	if err := t.run(ctx, args, info, func(f float64) { report("converting", f) }); err != nil {
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
