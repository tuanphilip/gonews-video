package tts

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

type SubtitleCue struct {
	Index     int
	StartTime float64 // seconds
	EndTime   float64 // seconds
	Text      string
}

func cleanVietnameseForEdge(s string) string {
	// Edge-TTS bug: phrases with standalone 'một' followed by certain nouns cause 400 NoAudioReceived.
	// Replacing standalone 'một' / 'Một' with '1' preserves 100% pronunciation while avoiding server socket drop.
	words := strings.Fields(s)
	for i, w := range words {
		lower := strings.ToLower(w)
		if lower == "một" {
			words[i] = "1"
		} else if strings.HasPrefix(lower, "một,") || strings.HasPrefix(lower, "một.") {
			words[i] = "1" + w[len("một"):]
		}
	}
	return strings.Join(words, " ")
}

// SynthesizeVoice tries edge-tts first; if that fails or edge-tts binary exists, fallback gracefully.
func SynthesizeVoice(text, voiceName, destAudioPath, destSRTPath string) ([]SubtitleCue, error) {
	text = cleanVietnameseForEdge(text)
	text = norm.NFC.String(strings.TrimSpace(text))
	if text == "" {
		return nil, fmt.Errorf("text is empty")
	}

	// 1. Try Vbee if voiceName belongs to Vbee
	if strings.HasPrefix(voiceName, "vbee:") || strings.HasPrefix(voiceName, "hn_") || strings.HasPrefix(voiceName, "sg_") {
		return synthesizeVbee(text, voiceName, destAudioPath, destSRTPath)
	}

	// 2. Try native edge-tts CLI tool if available in system or adjacent venv
	cliPaths := []string{
		"E:/Reverse/Video/MoneyPrinterTurbo/.venv/Scripts/python.exe",
		"python",
	}

	for _, py := range cliPaths {
		if fileExists(py) || py == "python" {
			// Write text to temp file to preserve exact UTF-8 NFC bytes across Windows CLI
			tmpTextFile := destAudioPath + ".txt"
			os.WriteFile(tmpTextFile, []byte(text), 0644)
			defer os.Remove(tmpTextFile)

			pyScript := fmt.Sprintf(`
import asyncio, edge_tts, unicodedata
async def main():
    with open(%q, 'r', encoding='utf-8') as f:
        raw = f.read().strip()
    text = unicodedata.normalize('NFC', raw)
    voice = %q
    comm = edge_tts.Communicate(text, voice, boundary='SentenceBoundary')
    submaker = edge_tts.SubMaker()
    with open(%q, 'wb') as f:
        async for chunk in comm.stream():
            if chunk['type'] == 'audio':
                f.write(chunk['data'])
            elif chunk['type'] in ('WordBoundary', 'SentenceBoundary'):
                submaker.feed(chunk)
    srt = submaker.get_srt()
    if srt:
        with open(%q, 'w', encoding='utf-8') as f:
            f.write(srt)
asyncio.run(main())
`, tmpTextFile, voiceName, destAudioPath, destSRTPath)

			cmd := exec.Command(py, "-c", pyScript)
			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("TTS Python (%s) failed: %v, out: %s\n", py, err, string(out))
			}
			if err == nil && fileExists(destAudioPath) && getFileSize(destAudioPath) > 0 {
				cues, _ := parseSRT(destSRTPath)
				if len(cues) > 0 {
					return cues, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("không tìm thấy công cụ TTS khả dụng")
}

func synthesizeVbee(text, voiceName, destAudioPath, destSRTPath string) ([]SubtitleCue, error) {
	apiURL := "https://api.vbee.vn/api/v1/tts"
	appID := "2f9e9876-4421-4112-b48e-a54229c05808"
	token := "eyJhbG...ZICU"

	reqPayload := map[string]interface{}{
		"app_id":     appID,
		"voice_code": voiceName,
		"input_text": text,
		"speed":      1.0,
		"audio_type": "mp3",
	}

	jsonBytes, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Status string `json:"status"`
		Result struct {
			AudioLink string `json:"audio_link"`
		} `json:"result"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &res); err != nil || res.Result.AudioLink == "" {
		return nil, fmt.Errorf("vbee error: %s", string(body))
	}

	if err := downloadFile(res.Result.AudioLink, destAudioPath); err != nil {
		return nil, err
	}

	duration := getAudioDuration(destAudioPath)
	cues := buildProportionalSRT(text, duration)
	writeSRT(cues, destSRTPath)
	return cues, nil
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func getFileSize(p string) int64 {
	info, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return info.Size()
}

func parseSRT(path string) ([]SubtitleCue, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	blocks := strings.Split(strings.TrimSpace(string(content)), "\n\n")
	var cues []SubtitleCue

	for i, b := range blocks {
		lines := strings.Split(strings.TrimSpace(b), "\n")
		if len(lines) >= 3 {
			cues = append(cues, SubtitleCue{
				Index: i + 1,
				Text:  strings.Join(lines[2:], " "),
			})
		}
	}
	return cues, nil
}

func buildProportionalSRT(text string, duration float64) []SubtitleCue {
	sentences := splitVietnameseSentences(text)
	if len(sentences) == 0 {
		sentences = []string{text}
	}
	totalChars := len([]rune(text))
	if totalChars == 0 {
		totalChars = 1
	}

	var cues []SubtitleCue
	curTime := 0.0
	for idx, s := range sentences {
		charCount := len([]rune(s))
		segDuration := duration * (float64(charCount) / float64(totalChars))
		cues = append(cues, SubtitleCue{
			Index:     idx + 1,
			StartTime: curTime,
			EndTime:   curTime + segDuration,
			Text:      s,
		})
		curTime += segDuration
	}
	return cues
}

func downloadFile(url string, dest string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func getAudioDuration(filePath string) float64 {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	out, err := cmd.Output()
	if err != nil {
		return 10.0
	}
	var d float64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &d)
	if d <= 0 {
		return 10.0
	}
	return d
}

func splitVietnameseSentences(text string) []string {
	var res []string
	raw := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '?' || r == '!' || r == '\n'
	})
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r != "" {
			res = append(res, r)
		}
	}
	return res
}

func writeSRT(cues []SubtitleCue, destPath string) error {
	os.MkdirAll(filepath.Dir(destPath), 0755)
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, c := range cues {
		start := formatSRTTime(c.StartTime)
		end := formatSRTTime(c.EndTime)
		fmt.Fprintf(f, "%d\n%s --> %s\n%s\n\n", i+1, start, end, c.Text)
	}
	return nil
}

func formatSRTTime(seconds float64) string {
	totalMs := int(seconds * 1000)
	hours := totalMs / 3600000
	totalMs %= 3600000
	mins := totalMs / 60000
	totalMs %= 60000
	secs := totalMs / 1000
	ms := totalMs % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, mins, secs, ms)
}
