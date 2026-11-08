package video

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type RenderOptions struct {
	Images       []string
	AudioPath    string
	SRTPath      string
	Headline1    string
	Headline2    string
	Headline     string
	FooterColor  string // Custom hex like "#EA580C", "#D97706", "#2563EB", "#0F172A", etc.
	TagText      string
	OutputPath   string
	Width        int
	Height       int
	FPS          int
	BGMVolume    float64
	VoiceVolume  float64
}

func RenderNewsVideo(opt RenderOptions) error {
	if len(opt.Images) == 0 {
		return fmt.Errorf("no images provided for video render")
	}
	if opt.Width == 0 {
		opt.Width = 1080
	}
	if opt.Height == 0 {
		opt.Height = 1920
	}
	if opt.FPS == 0 {
		opt.FPS = 30
	}

	// 1. Get audio duration
	audioDuration := getDuration(opt.AudioPath)
	if audioDuration <= 0 {
		audioDuration = 15.0
	}

	// Target duration: audio + 0.5s padding
	totalDuration := audioDuration + 0.5
	numImages := len(opt.Images)
	clipDuration := totalDuration / float64(numImages)
	clipFrames := int(math.Ceil(clipDuration * float64(opt.FPS)))

	// 2. Build FFmpeg filter_complex for image slideshow (zoompan)
	var filterInputs []string
	var filterGraph strings.Builder

	for i, img := range opt.Images {
		filterInputs = append(filterInputs, "-loop", "1", "-t", fmt.Sprintf("%.2f", clipDuration), "-i", img)
		// Ken Burns alternate zoom in / zoom out
		zoomExpr := "min(zoom+0.0015,1.25)"
		if i%2 == 1 {
			zoomExpr = "if(eq(on,1),1.25,max(1.0,zoom-0.0015))"
		}
		// Scale image to fill viewport and zoompan smoothly
		filterGraph.WriteString(fmt.Sprintf(
			"[%d:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,zoompan=z='%s':x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)':d=%d:s=%dx%d:fps=%d,setsar=1[v%d];",
			i, opt.Width, opt.Height, opt.Width, opt.Height, zoomExpr, clipFrames, opt.Width, opt.Height, opt.FPS, i,
		))
	}

	// Concat video streams
	for i := 0; i < numImages; i++ {
		filterGraph.WriteString(fmt.Sprintf("[v%d]", i))
	}
	filterGraph.WriteString(fmt.Sprintf("concat=n=%d:v=1:a=0[vbase];", numImages))

	// 3. News Fixed Footer Overlay (Theanh28/Beatvn style)
	// Color parsing
	footerHex := parseHexColor(opt.FooterColor, "0xEA580C")
	tagBgHex := adjustBrightness(footerHex, 0.7)

	tagText := opt.TagText
	if tagText == "" {
		tagText = "● TIN NÓNG HÔM NAY"
	}

	h1 := opt.Headline1
	h2 := opt.Headline2
	if h1 == "" && h2 == "" && opt.Headline != "" {
		// Split by hyphen or comma if exists
		parts := strings.Split(opt.Headline, " - ")
		if len(parts) >= 2 {
			h1 = parts[0]
			h2 = parts[1]
		} else {
			h1 = opt.Headline
		}
	}
	cleanH1 := escapeDrawText(strings.ToUpper(h1))
	cleanH2 := escapeDrawText(strings.ToUpper(h2))

	// Find bundled Vietnamese font
	fontFile := "E:/Reverse/Video/MoneyPrinterTurbo/resource/fonts/BeVietnamPro-Bold.ttf"
	escapedFont := strings.ReplaceAll(fontFile, "\\", "/")
	escapedFont = strings.ReplaceAll(escapedFont, ":", "\\:")

	// Footer starts at Y = 1152 (60% video, height = 768px for 1080x1920)
	footerY := int(float64(opt.Height) * 0.60)
	footerH := opt.Height - footerY

	filterGraph.WriteString(fmt.Sprintf(
		"[vbase]drawbox=y=%d:w=%d:h=%d:color=%s:t=fill[vbg];",
		footerY, opt.Width, footerH, footerHex,
	))

	// Badge Tag
	tagY := footerY + 36
	filterGraph.WriteString(fmt.Sprintf(
		"[vbg]drawbox=x=60:y=%d:w=280:h=46:color=%s:t=fill[vtagbox];",
		tagY, tagBgHex,
	))
	filterGraph.WriteString(fmt.Sprintf(
		"[vtagbox]drawtext=fontfile='%s':text='%s':fontcolor=white:fontsize=22:x=85:y=%d[vtag];",
		escapedFont, escapeDrawText(tagText), tagY+11,
	))

	// Headline 1 (White Bold)
	h1Y := tagY + 70
	filterGraph.WriteString(fmt.Sprintf(
		"[vtag]drawtext=fontfile='%s':text='%s':fontcolor=white:fontsize=40:x=60:y=%d[vh1];",
		escapedFont, cleanH1, h1Y,
	))

	// Headline 2 (Yellow Bold - Key phrase)
	lastStream := "vh1"
	if cleanH2 != "" {
		h2Y := h1Y + 90
		filterGraph.WriteString(fmt.Sprintf(
			"[vh1]drawtext=fontfile='%s':text='%s':fontcolor=0xFEF08A:fontsize=40:x=60:y=%d[vh2];",
			escapedFont, cleanH2, h2Y,
		))
		lastStream = "vh2"
	}

	// Subtitle Burn-in (placed neatly in the lower safe zone)
	if opt.SRTPath != "" {
		escapedSRT := strings.ReplaceAll(opt.SRTPath, "\\", "/")
		escapedSRT = strings.ReplaceAll(escapedSRT, ":", "\\:")
		subFilter := fmt.Sprintf("subtitles='%s':force_style='FontSize=26,FontName=Arial,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,BackColour=&H80000000,Outline=2,Shadow=1,MarginV=120,Alignment=2'", escapedSRT)
		filterGraph.WriteString(fmt.Sprintf("[%s]%s[vfinal]", lastStream, subFilter))
		lastStream = "vfinal"
	}

	// Prepare ffmpeg arguments
	args := []string{"-y"}
	args = append(args, filterInputs...)
	args = append(args, "-i", opt.AudioPath)
	audioIndex := numImages

	args = append(args,
		"-filter_complex", filterGraph.String(),
		"-map", fmt.Sprintf("[%s]", lastStream),
		"-map", fmt.Sprintf("%d:a", audioIndex),
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "22",
		"-c:a", "aac",
		"-b:a", "192k",
		"-shortest",
		"-pix_fmt", "yuv420p",
		opt.OutputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, log:\n%s", err, string(output))
	}

	return nil
}

func parseHexColor(hexStr, defaultHex string) string {
	hexStr = strings.TrimSpace(hexStr)
	if hexStr == "" {
		return defaultHex
	}
	hexStr = strings.TrimPrefix(hexStr, "#")
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		return hexStr
	}
	if len(hexStr) == 6 {
		return "0x" + strings.ToUpper(hexStr)
	}
	return defaultHex
}

func adjustBrightness(hexStr string, factor float64) string {
	clean := strings.TrimPrefix(hexStr, "0x")
	clean = strings.TrimPrefix(clean, "0X")
	if len(clean) != 6 {
		return hexStr
	}
	var r, g, b int
	fmt.Sscanf(clean, "%02x%02x%02x", &r, &g, &b)
	r = int(float64(r) * factor)
	g = int(float64(g) * factor)
	b = int(float64(b) * factor)
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}
	return fmt.Sprintf("0x%02X%02X%02X", r, g, b)
}

func escapeDrawText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "'", "'\\\\\\''")
	text = strings.ReplaceAll(text, ":", "\\:")
	text = strings.ReplaceAll(text, "%", "%%")
	return text
}

func getDuration(path string) float64 {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0
	}
	return d
}
