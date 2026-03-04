package channel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const audioDurationFallbackMillis = 1000

func localAevitasBinary(name string) string {
	return filepath.Join(os.Getenv("HOME"), ".aevitas", "bin", name)
}

func resolveLocalOrSystemBinary(name string) (string, error) {
	local := localAevitasBinary(name)
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}
	system, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s not found at %s and not found in PATH", name, local)
	}
	return system, nil
}

func transcodeToTelegramVoice(srcPath string) (string, error) {
	ffmpegPath, err := resolveLocalOrSystemBinary("ffmpeg")
	if err != nil {
		return "", err
	}
	tempDir := filepath.Join(os.TempDir(), "aevitas-telegram-media")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	dstPath := filepath.Join(tempDir, fmt.Sprintf("voice-%d.ogg", time.Now().UnixNano()))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-y",
		"-i", srcPath,
		"-c:a", "libopus",
		"-b:a", "48k",
		"-vbr", "on",
		"-compression_level", "10",
		"-frame_duration", "60",
		"-application", "voip",
		dstPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("ffmpeg transcode failed: %v output=%s", err, strings.TrimSpace(string(out)))
	}
	return dstPath, nil
}

func transcodeToFeishuOpus(srcPath string) (string, error) {
	ffmpegPath, err := resolveLocalOrSystemBinary("ffmpeg")
	if err != nil {
		return "", err
	}
	tempDir := filepath.Join(os.TempDir(), "aevitas-feishu-media")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	dstPath := filepath.Join(tempDir, fmt.Sprintf("audio-%d.opus", time.Now().UnixNano()))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-y",
		"-i", srcPath,
		"-c:a", "libopus",
		"-b:a", "48k",
		"-vbr", "on",
		"-compression_level", "10",
		"-application", "voip",
		dstPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("ffmpeg opus transcode failed: %v output=%s", err, strings.TrimSpace(string(out)))
	}
	return dstPath, nil
}

func detectAudioDurationMillis(mediaPath string) int {
	ffprobePath, err := resolveLocalOrSystemBinary("ffprobe")
	if err != nil {
		return audioDurationFallbackMillis
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		mediaPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return audioDurationFallbackMillis
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || seconds <= 0 {
		return audioDurationFallbackMillis
	}
	ms := int(seconds * 1000)
	if ms < audioDurationFallbackMillis {
		return audioDurationFallbackMillis
	}
	return ms
}
