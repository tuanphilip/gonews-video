package main

import (
	"fmt"
	"log"
	"os"

	"gonews-video/internal/config"
	"gonews-video/internal/crawler"
	"gonews-video/internal/summarizer"
	"gonews-video/internal/tts"
	"gonews-video/internal/video"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_pipeline.go <article_url>")
		return
	}

	targetURL := os.Args[1]
	cfg := config.DefaultConfig()
	workDir := "./test_run"
	os.MkdirAll(workDir, 0755)

	log.Println("[1/4] Crawling article:", targetURL)
	article, err := crawler.CrawlArticle(targetURL, workDir)
	if err != nil {
		log.Fatalf("Crawl failed: %v", err)
	}
	log.Printf("Found title: %s (%d images downloaded)\n", article.Title, len(article.LocalImages))
	if len(article.LocalImages) == 0 {
		log.Fatalf("No images found in article")
	}

	log.Println("[2/4] Summarizing with 9router (Gemini 3.8 Flash)...")
	script, err := summarizer.GenerateNewsScript(cfg, article.Title, article.Content)
	if err != nil {
		log.Fatalf("Summarize failed: %v", err)
	}
	log.Printf("Headline: %s\nScript: %s\n", script.Headline, script.FullScript)

	log.Println("[3/4] Synthesizing Voiceover with Edge-TTS (vi-VN-HoaiMyNeural)...")
	audioPath := workDir + "/voiceover.mp3"
	srtPath := workDir + "/subtitles.srt"
	_, err = tts.SynthesizeVoice(script.FullScript, "vi-VN-HoaiMyNeural", audioPath, srtPath)
	if err != nil {
		log.Fatalf("TTS failed: %v", err)
	}
	log.Println("Audio & Subtitle generated successfully.")

	log.Println("[4/4] Rendering video with Native FFmpeg FilterGraph...")
	outPath := workDir + "/final_news.mp4"
	err = video.RenderNewsVideo(video.RenderOptions{
		Images:      article.LocalImages,
		AudioPath:   audioPath,
		SRTPath:     srtPath,
		Headline1:   script.Headline1,
		Headline2:   script.Headline2,
		Headline:    script.Headline,
		FooterColor: "#EA580C",
		TagText:     "● TIN NÓNG HÔM NAY",
		OutputPath:  outPath,
		Width:       1080,
		Height:      1920,
		FPS:         30,
	})
	if err != nil {
		log.Fatalf("Render failed: %v", err)
	}

	log.Println("SUCCESS! Final news video:", outPath)
}
