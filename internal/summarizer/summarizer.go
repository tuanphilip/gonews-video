package summarizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gonews-video/internal/config"
)

type NewsScript struct {
	Headline1  string   `json:"headline1"`
	Headline2  string   `json:"headline2"`
	Headline   string   `json:"headline"`
	FullScript string   `json:"fullScript"`
	Sentences  []string `json:"sentences"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func GenerateNewsScript(cfg config.Config, title, content string) (*NewsScript, error) {
	prompt := fmt.Sprintf(`Bạn là một biên tập viên giật tít tin tức video ngắn TikTok/Reels chuyên nghiệp (phong cách Theanh28, Beatvn).
Hãy đọc nội dung bài báo sau và tạo tiêu đề 2 dòng giật gân, khơi gợi tò mò, cùng kịch bản bản tin thời sự cô đọng:

Tiêu đề gốc: %s
Nội dung bài báo:
%s

Yêu cầu kỹ thuật BẮT BUỘC:
1. headline1: Dòng 1 ngắn gọn, giật tít mạnh, viết hoa (tối đa 8 từ, VD: TIN VUI CỰC MẠNH CHO HÀNG TRIỆU SINH VIÊN).
2. headline2: Dòng 2 chứa sự việc bất ngờ/từ khóa quan trọng, viết hoa (tối đa 9 từ, VD: CHÍNH THỨC CẤM ĐẠI HỌC ÉP NỘP TIỀN CẢ KỲ).
3. fullScript: Lời bình đọc thành mạch liên tục từ 20 đến 35 giây, dứt khoát, giữ đúng số liệu. QUY TẮC PHÁT ÂM: Thay vì viết "một", hãy dùng số "1" (ví dụ: "1 năm", "1 thông tin") để engine đọc trơn tru nhất.
4. Trả về DUY NHẤT một chuỗi JSON hợp lệ với cấu trúc sau, không thêm markdown hay giải thích nào khác:
{
  "headline1": "DÒNG 1 VIẾT HOA",
  "headline2": "DÒNG 2 VIẾT HOA",
  "fullScript": "Toàn bộ nội dung lời bình đọc thành mạch liên tục.",
  "sentences": [
    "Câu 1 mở đầu.",
    "Câu 2 diễn biến.",
    "Câu 3 chi tiết.",
    "Câu 4 kết luận."
  ]
}`, title, content)

	reqBody := chatRequest{
		Model: cfg.NineRouterModel,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		Stream:      false,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	apiURL := strings.TrimRight(cfg.NineRouterURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.NineRouterKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	// Handle SSE stream fallback if proxy ignores stream=false
	if strings.HasPrefix(strings.TrimSpace(string(body)), "data:") {
		var contentBuilder strings.Builder
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				dataPayload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataPayload == "[DONE]" || dataPayload == "" {
					continue
				}
				var chunk struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(dataPayload), &chunk); err == nil && len(chunk.Choices) > 0 {
					contentBuilder.WriteString(chunk.Choices[0].Delta.Content)
				}
			}
		}
		rawContent := strings.TrimSpace(contentBuilder.String())
		rawContent = strings.TrimPrefix(rawContent, "```json")
		rawContent = strings.TrimPrefix(rawContent, "```")
		rawContent = strings.TrimSuffix(rawContent, "```")
		rawContent = strings.TrimSpace(rawContent)

		var script NewsScript
		if err := json.Unmarshal([]byte(rawContent), &script); err != nil {
			return nil, fmt.Errorf("failed to parse SSE script json: %w (raw: %s)", err, rawContent)
		}
		if script.Headline == "" {
			if script.Headline1 != "" && script.Headline2 != "" {
				script.Headline = script.Headline1 + " - " + script.Headline2
			} else {
				script.Headline = script.Headline1
			}
		}
		return &script, nil
	}

	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, raw body: %s", err, string(body))
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices returned from llm")
	}

	rawContent := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Remove markdown block if model wraps JSON
	rawContent = strings.TrimPrefix(rawContent, "```json")
	rawContent = strings.TrimPrefix(rawContent, "```")
	rawContent = strings.TrimSuffix(rawContent, "```")
	rawContent = strings.TrimSpace(rawContent)

	var script NewsScript
	if err := json.Unmarshal([]byte(rawContent), &script); err != nil {
		// Fallback parse if JSON is slightly off
		return nil, fmt.Errorf("failed to parse script json: %w (raw: %s)", err, rawContent)
	}

	if script.Headline == "" {
		if script.Headline1 != "" && script.Headline2 != "" {
			script.Headline = script.Headline1 + " - " + script.Headline2
		} else {
			script.Headline = script.Headline1
		}
	}

	if len(script.Sentences) == 0 && script.FullScript != "" {
		script.Sentences = []string{script.FullScript}
	}

	return &script, nil
}
