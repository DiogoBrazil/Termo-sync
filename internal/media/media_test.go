package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Set DEPOIMENTO_TEST_FFMPEG and DEPOIMENTO_TEST_FFPROBE to run real synthetic-media tests.
func TestSyntheticMedia(t *testing.T) {
	ffmpeg, ffprobe := os.Getenv("DEPOIMENTO_TEST_FFMPEG"), os.Getenv("DEPOIMENTO_TEST_FFPROBE")
	if ffmpeg == "" || ffprobe == "" {
		t.Skip("configure ferramentas para testes reais de mídia")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	tools := Tools{FFmpeg: ffmpeg, FFprobe: ffprobe}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	generate := func(name string, args ...string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		args = append([]string{"-nostdin", "-v", "error"}, args...)
		args = append(args, path)
		if data, err := exec.CommandContext(ctx, ffmpeg, args...).CombinedOutput(); err != nil {
			t.Fatalf("generate: %v %s", err, data)
		}
		return path
	}
	mp3 := generate("áudio com espaço ' $ teste.mp3", "-f", "lavfi", "-i", "sine=frequency=400:duration=0.3", "-c:a", "libmp3lame")
	info, err := tools.Inspect(ctx, mp3)
	if err != nil {
		t.Fatal(err)
	}
	direct, clean, err := tools.Prepare(ctx, mp3, info, info.Streams[0].Index)
	if err != nil {
		t.Fatal(err)
	}
	clean()
	if direct != mp3 {
		t.Fatal("valid MP3 rewritten")
	}
	if _, err := os.Stat(mp3); err != nil {
		t.Fatal("original removed")
	}
	video := generate("fictício.mp4", "-f", "lavfi", "-i", "color=size=32x32:duration=0.3", "-f", "lavfi", "-i", "sine=frequency=400:duration=0.3", "-c:v", "mpeg4", "-c:a", "aac", "-shortest")
	info, err = tools.Inspect(ctx, video)
	if err != nil {
		t.Fatal(err)
	}
	output, cleanup, err := tools.Prepare(ctx, video, info, info.Streams[0].Index)
	if err != nil {
		t.Fatal(err)
	}
	if output == video {
		t.Fatal("video original overwritten")
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
	cleanup()
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("temporary leaked")
	}
	multi := generate("duas-trilhas.mkv", "-f", "lavfi", "-i", "sine=frequency=400:duration=0.3", "-f", "lavfi", "-i", "sine=frequency=800:duration=0.3", "-map", "0:a", "-map", "1:a", "-c:a", "pcm_s16le")
	info, err = tools.Inspect(ctx, multi)
	if err != nil || len(info.Streams) != 2 {
		t.Fatalf("ambiguous tracks lost: %v", err)
	}
	if _, cleanup, err := tools.Prepare(ctx, multi, info, 999); err == nil {
		cleanup()
		t.Fatal("unknown track")
	}
	silent := generate("sem-audio.mp4", "-f", "lavfi", "-i", "color=size=32x32:duration=0.3", "-c:v", "mpeg4")
	if _, err := tools.Inspect(ctx, silent); err == nil {
		t.Fatal("video without audio accepted")
	}
	corrupt := filepath.Join(dir, "corrupto.mp3")
	if err := os.WriteFile(corrupt, []byte("not audio"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := tools.Inspect(ctx, corrupt); err == nil {
		t.Fatal("corrupt media accepted")
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err := tools.Inspect(cancelled, mp3); err == nil {
		t.Fatal("cancel ignored")
	}
	// Force conversion failure and assert cleanup left no session directory.
	info, err = tools.Inspect(ctx, video)
	if err != nil {
		t.Fatal(err)
	}
	failing := Tools{FFmpeg: filepath.Join(dir, "missing-ffmpeg"), FFprobe: ffprobe}
	if _, cleanup, err := failing.Prepare(ctx, video, info, info.Streams[0].Index); err == nil {
		cleanup()
		t.Fatal("missing executable accepted")
	}
	root, err := tempRoot()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("failed conversion left temporary files")
	}
	oldDir := filepath.Join(root, "session-stale")
	freshDir := filepath.Join(root, "session-fresh")
	for _, d := range []string{oldDir, freshDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(oldDir, past, past); err != nil {
		t.Fatal(err)
	}
	if err := Recover(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatal("stale session directory was not recovered")
	}
	if _, err := os.Stat(freshDir); err != nil {
		t.Fatal("fresh session directory was removed prematurely")
	}
}
