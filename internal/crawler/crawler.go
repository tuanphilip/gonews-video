package crawler

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Article struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	ImageURLs   []string `json:"imageUrls"`
	LocalImages []string `json:"localImages"`
}

var client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func CrawlArticle(targetURL string, outputDir string) (*Article, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html error: %w", err)
	}

	article := &Article{URL: targetURL}

	// 1. Extract Title
	article.Title = strings.TrimSpace(doc.Find("meta[property='og:title']").AttrOr("content", ""))
	if article.Title == "" {
		article.Title = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if article.Title == "" {
		article.Title = strings.TrimSpace(doc.Find("title").Text())
	}

	// 2. Extract Description
	article.Description = strings.TrimSpace(doc.Find("meta[property='og:description']").AttrOr("content", ""))
	if article.Description == "" {
		article.Description = strings.TrimSpace(doc.Find("meta[name='description']").AttrOr("content", ""))
	}

	// 3. Extract Body paragraphs
	var paragraphs []string
	doc.Find("article p, .fck_detail p, .content p, .article-content p, .detail-content p, .story-body p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Skip short metadata / caption / copyright notices
		if len([]rune(text)) > 20 && !strings.HasPrefix(text, "Ảnh:") && !strings.HasPrefix(text, "Video:") {
			paragraphs = append(paragraphs, text)
		}
	})
	article.Content = strings.Join(paragraphs, "\n\n")

	// 4. Extract Images
	imageSet := make(map[string]bool)
	doc.Find("article img, .fck_detail img, .content img, .article-content img, .detail-content img").Each(func(i int, s *goquery.Selection) {
		src := s.AttrOr("data-src", "")
		if src == "" {
			src = s.AttrOr("data-original", "")
		}
		if src == "" {
			src = s.AttrOr("src", "")
		}
		src = strings.TrimSpace(src)
		if src == "" || strings.HasPrefix(src, "data:") {
			return
		}

		// Resolve relative URL
		resolvedURL := resolveURL(targetURL, src)
		lower := strings.ToLower(resolvedURL)
		if strings.Contains(lower, "icon") || strings.Contains(lower, "logo") || strings.Contains(lower, "avatar") || strings.Contains(lower, "badge") {
			return
		}
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") || strings.Contains(lower, "/images/") || strings.Contains(lower, "vnecdn.net") {
			if !imageSet[resolvedURL] {
				imageSet[resolvedURL] = true
				article.ImageURLs = append(article.ImageURLs, resolvedURL)
			}
		}
	})

	// Download up to 6 images
	os.MkdirAll(outputDir, 0755)
	maxImages := 6
	if len(article.ImageURLs) < maxImages {
		maxImages = len(article.ImageURLs)
	}

	for i := 0; i < maxImages; i++ {
		imgURL := article.ImageURLs[i]
		ext := ".jpg"
		if strings.HasSuffix(strings.ToLower(imgURL), ".png") {
			ext = ".png"
		}
		destPath := filepath.Join(outputDir, fmt.Sprintf("news_img_%d%s", i+1, ext))
		if err := downloadFile(imgURL, destPath); err == nil {
			article.LocalImages = append(article.LocalImages, destPath)
		}
	}

	return article, nil
}

func resolveURL(baseURL, relURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return relURL
	}
	rel, err := url.Parse(relURL)
	if err != nil {
		return relURL
	}
	return base.ResolveReference(rel).String()
}

func downloadFile(url string, dest string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
