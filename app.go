package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gonews-video/internal/config"
	"gonews-video/internal/crawler"
	"gonews-video/internal/summarizer"
	"gonews-video/internal/tts"
	"gonews-video/internal/video"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	cfg config.Config
}

type ProcessProgress struct {
	Step       string `json:"step"`
	Percent    int    `json:"percent"`
	Log        string `json:"log"`
	IsComplete bool   `json:"isComplete"`
	Error      string `json:"error,omitempty"`
	VideoPath  string `json:"videoPath,omitempty"`
}

type ArticlePreview struct {
	TaskID     string   `json:"taskId"`
	URL        string   `json:"url"`
	Title      string   `json:"title"`
	Headline1  string   `json:"headline1"`
	Headline2  string   `json:"headline2"`
	Headline   string   `json:"headline"`
	FullScript string   `json:"fullScript"`
	Sentences  []string `json:"sentences"`
	Images     []string `json:"images"`
}

type RenderRequest struct {
	TaskID      string   `json:"taskId"`
	Headline1   string   `json:"headline1"`
	Headline2   string   `json:"headline2"`
	Headline    string   `json:"headline"`
	FooterColor string   `json:"footerColor"`
	TagText     string   `json:"tagText"`
	Script      string   `json:"script"`
	Images      []string `json:"images"`
	VoiceName   string   `json:"voiceName"`
	Ratio       string   `json:"ratio"`
}

func NewApp() *App {
	return &App{
		cfg: config.DefaultConfig(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetConfig() config.Config {
	return a.cfg
}

func (a *App) SaveConfig(cfg config.Config) bool {
	a.cfg = cfg
	return true
}

func (a *App) OpenFolder(filePath string) bool {
	if filePath == "" {
		filePath = a.cfg.OutputDir
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	cmd := exec.Command("explorer.exe", "/select,", absPath)
	if err := cmd.Start(); err != nil {
		exec.Command("explorer.exe", filepath.Dir(absPath)).Start()
	}
	return true
}

func (a *App) PreviewArticle(targetURL string) (*ArticlePreview, error) {
	taskID := fmt.Sprintf("task_%d", time.Now().Unix())
	taskDir := filepath.Join(a.cfg.OutputDir, taskID)
	os.MkdirAll(taskDir, 0755)

	runtime.EventsEmit(a.ctx, "pipeline:progress", ProcessProgress{
		Step:    "crawling",
		Percent: 20,
		Log:     "Đang bóc tách bài báo và tải hình ảnh chất lượng cao...",
	})

	article, err := crawler.CrawlArticle(targetURL, taskDir)
	if err != nil {
		return nil, fmt.Errorf("cào bài báo thất bại: %w", err)
	}

	runtime.EventsEmit(a.ctx, "pipeline:progress", ProcessProgress{
		Step:    "summarizing",
		Percent: 60,
		Log:     "AI 9router đang biên tập kịch bản tin tức thời sự...",
	})

	script, err := summarizer.GenerateNewsScript(a.cfg, article.Title, article.Content)
	if err != nil {
		return nil, fmt.Errorf("biên tập kịch bản thất bại: %w", err)
	}

	runtime.EventsEmit(a.ctx, "pipeline:progress", ProcessProgress{
		Step:       "preview_ready",
		Percent:    100,
		Log:        "Bóc tách thành công! Bạn có thể kiểm tra và chỉnh sửa trước khi render.",
		IsComplete: true,
	})

	var images []string
	for _, img := range article.LocalImages {
		abs, _ := filepath.Abs(img)
		images = append(images, abs)
	}

	return &ArticlePreview{
		TaskID:     taskID,
		URL:        targetURL,
		Title:      article.Title,
		Headline1:  script.Headline1,
		Headline2:  script.Headline2,
		Headline:   script.Headline,
		FullScript: script.FullScript,
		Sentences:  script.Sentences,
		Images:     images,
	}, nil
}

func (a *App) RenderFromPreview(req RenderRequest) (*ProcessProgress, error) {
	if req.VoiceName == "" {
		req.VoiceName = "vi-VN-HoaiMyNeural"
	}
	if req.TaskID == "" {
		req.TaskID = fmt.Sprintf("task_%d", time.Now().Unix())
	}
	taskDir := filepath.Join(a.cfg.OutputDir, req.TaskID)
	os.MkdirAll(taskDir, 0755)

	emitProgress := func(step string, percent int, log string) {
		runtime.EventsEmit(a.ctx, "pipeline:progress", ProcessProgress{
			Step:    step,
			Percent: percent,
			Log:     log,
		})
	}

	if len(req.Images) == 0 {
		return nil, fmt.Errorf("cần ít nhất 1 hình ảnh để dựng video")
	}

	// 1. Synthesize audio & srt
	emitProgress("tts", 35, fmt.Sprintf("Đang lồng tiếng thời sự (%s)...", req.VoiceName))
	audioPath := filepath.Join(taskDir, "voiceover.mp3")
	srtPath := filepath.Join(taskDir, "subtitles.srt")

	_, err := tts.SynthesizeVoice(req.Script, req.VoiceName, audioPath, srtPath)
	if err != nil {
		return nil, fmt.Errorf("lồng tiếng thất bại: %w", err)
	}

	// 2. Render FFmpeg
	emitProgress("rendering", 75, "Đang dựng video HD chuyển động & phụ đề karaoke...")
	finalVideoPath := filepath.Join(taskDir, "final_news.mp4")

	width, height := 1080, 1920
	if req.Ratio == "16:9" {
		width, height = 1920, 1080
	}

	renderOpt := video.RenderOptions{
		Images:      req.Images,
		AudioPath:   audioPath,
		SRTPath:     srtPath,
		Headline1:   req.Headline1,
		Headline2:   req.Headline2,
		Headline:    req.Headline,
		FooterColor: req.FooterColor,
		TagText:     req.TagText,
		OutputPath:  finalVideoPath,
		Width:       width,
		Height:      height,
		FPS:         30,
		BGMVolume:   0.2,
		VoiceVolume: 1.0,
	}

	if err := video.RenderNewsVideo(renderOpt); err != nil {
		return nil, fmt.Errorf("render video thất bại: %w", err)
	}

	emitProgress("complete", 100, "Xuất video thành công!")

	absVideoPath, _ := filepath.Abs(finalVideoPath)
	return &ProcessProgress{
		Step:       "complete",
		Percent:    100,
		Log:        "Xuất video hoàn tất!",
		IsComplete: true,
		VideoPath:  absVideoPath,
	}, nil
}

func (a *App) ProcessArticleToVideo(targetURL string, voiceName string, customHeadline string, footerColor string) (*ProcessProgress, error) {
	prev, err := a.PreviewArticle(targetURL)
	if err != nil {
		return nil, err
	}
	h1 := prev.Headline1
	h2 := prev.Headline2
	if customHeadline != "" {
		h1 = customHeadline
		h2 = ""
	}
	if footerColor == "" {
		footerColor = "#EA580C"
	}
	return a.RenderFromPreview(RenderRequest{
		TaskID:      prev.TaskID,
		Headline1:   h1,
		Headline2:   h2,
		Headline:    prev.Headline,
		FooterColor: footerColor,
		TagText:     "● TIN NÓNG HÔM NAY",
		Script:      prev.FullScript,
		Images:      prev.Images,
		VoiceName:   voiceName,
		Ratio:       "9:16",
	})
}
