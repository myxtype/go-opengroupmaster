package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"supervisor/internal/config"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type spamAIInput struct {
	Content string
}

type spamAIResult struct {
	Label  string `json:"label"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

func (r spamAIResult) IsSpamBy(threshold int) bool {
	label := strings.ToLower(strings.TrimSpace(r.Label))
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	return label == "spam" && r.Score >= threshold
}

func (r spamAIResult) Normalized() (spamAIResult, error) {
	out := r
	out.Label = strings.ToLower(strings.TrimSpace(out.Label))
	switch out.Label {
	case "spam", "ham":
	default:
		return spamAIResult{}, fmt.Errorf("invalid ai label: %q", out.Label)
	}
	if out.Score < 0 {
		out.Score = 0
	}
	if out.Score > 100 {
		out.Score = 100
	}
	out.Reason = strings.TrimSpace(out.Reason)
	if out.Reason == "" {
		if out.Label == "spam" {
			out.Reason = "ai spam judgement"
		} else {
			out.Reason = "ai ham judgement"
		}
	}
	if len(out.Reason) > 120 {
		out.Reason = out.Reason[:120]
	}
	return out, nil
}

func parseSpamAIResult(raw string) (spamAIResult, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return spamAIResult{}, errors.New("empty ai output")
	}
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start < 0 || end < start {
		return spamAIResult{}, fmt.Errorf("ai output is not json: %q", clean)
	}
	payload := clean[start : end+1]
	var out spamAIResult
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return spamAIResult{}, err
	}
	return out.Normalized()
}

type spamAIClassifier interface {
	Classify(ctx context.Context, input spamAIInput) (spamAIResult, error)
	Name() string
}

func newSpamAIClassifier(cfg *config.Config, logger *log.Logger) spamAIClassifier {
	c, err := newLangChainOllamaClassifier(cfg.AntiSpamAIModel, cfg.AntiSpamAIServerURL, time.Duration(cfg.AntiSpamAITimeoutSecs)*time.Second, logger)
	if err != nil {
		panic(err)
	}
	logger.Printf("anti spam ai ready: provider=%s model=%s server=%s timeout=%ds", c.Name(), cfg.AntiSpamAIModel, cfg.AntiSpamAIServerURL, cfg.AntiSpamAITimeoutSecs)
	return c
}

type langChainOllamaClassifier struct {
	llm     llms.Model
	timeout time.Duration
}

func newLangChainOllamaClassifier(model, serverURL string, timeout time.Duration, logger *log.Logger) (spamAIClassifier, error) {
	_ = logger
	client, err := ollama.New(
		ollama.WithModel(model),
		ollama.WithServerURL(serverURL),
	)
	if err != nil {
		return nil, err
	}
	return &langChainOllamaClassifier{
		llm:     client,
		timeout: timeout,
	}, nil
}

func (c *langChainOllamaClassifier) Name() string {
	return "langchaingo_ollama"
}

func (c *langChainOllamaClassifier) Classify(ctx context.Context, input spamAIInput) (spamAIResult, error) {
	if c == nil || c.llm == nil {
		return spamAIResult{}, fmt.Errorf("nil ai classifier")
	}
	callCtx := ctx
	cancel := func() {}
	if c.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	defer cancel()

	raw, err := llms.GenerateFromSinglePrompt(
		callCtx,
		c.llm,
		buildSpamAIPrompt(input),
		llms.WithTemperature(0),
		llms.WithMaxTokens(120),
	)
	if err != nil {
		return spamAIResult{}, err
	}
	parsed, err := parseSpamAIResult(raw)
	if err != nil {
		return spamAIResult{}, err
	}
	return parsed, nil
}

func buildSpamAIPrompt(input spamAIInput) string {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		content = "(empty)"
	}
	var b strings.Builder
	b.WriteString("你是群聊反垃圾二分类器，主要识别广告、诱导、诈骗、色情、政治等垃圾信息。")
	b.WriteString("\n仅输出 JSON，不要输出 Markdown、解释、代码块。")
	b.WriteString("\n字段固定: {\"label\":\"spam|ham\",\"score\":0-100,\"reason\":\"短原因\"}")
	b.WriteString("\n规则：")
	b.WriteString("\n1) label 只能是 spam 或 ham")
	b.WriteString("\n2) score 必须是整数")
	b.WriteString("\n3) reason 要短，不超过 20 个字")
	b.WriteString("\n4) 严禁输出任何多余字段")
	b.WriteString("\n")
	b.WriteString("\n消息内容如下：")
	b.WriteString("\n<<<")
	b.WriteString(content)
	b.WriteString("\n>>>")
	b.WriteString("\n")
	b.WriteString("\n请直接输出 JSON：")
	return b.String()
}
