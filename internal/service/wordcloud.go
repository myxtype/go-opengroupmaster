package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"supervisor/internal/model"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/psykhi/wordclouds"
	"github.com/yanyiwu/gojieba"
)

const (
	wordCloudWordLimit       = 160
	wordCloudDrawWordLimit   = 100
	wordCloudContributorSize = 10
)

var wordCloudJiebaDictFiles = []string{
	"jieba.dict.utf8",
	"hmm_model.utf8",
	"user.dict.utf8",
	"idf.utf8",
	"stop_words.utf8",
}

var (
	wordCloudNonWordPattern = regexp.MustCompile(`^[\p{P}\p{S}\p{Zs}]+$`)
	wordCloudTimePattern    = regexp.MustCompile(`^(\d{1,2}):(\d{1,2})$`)
)

var wordCloudDefaultStopWords = map[string]struct{}{
	"的": {}, "了": {}, "和": {}, "是": {}, "就": {}, "都": {}, "而": {}, "及": {}, "与": {}, "着": {}, "或": {}, "一个": {},
	"我们": {}, "你们": {}, "他们": {}, "她们": {}, "这个": {}, "那个": {}, "这样": {}, "那样": {}, "以及": {},
	"the": {}, "and": {}, "for": {}, "that": {}, "this": {}, "with": {}, "are": {}, "from": {}, "you": {}, "your": {}, "http": {}, "https": {}, "www": {},
}

func wordCloudDayKey(t time.Time) string {
	return t.In(time.Local).Format("2006-01-02")
}

func (s *Service) WordCloudPanelViewByTGGroupID(tgGroupID int64) (*WordCloudPanelView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureWordCloud, false)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getWordCloudConfig(group.ID)
	if err != nil {
		return nil, err
	}
	blacklistCount, err := s.repo.CountWordCloudBlacklistWords(group.ID)
	if err != nil {
		return nil, err
	}
	return &WordCloudPanelView{
		Enabled:        enabled,
		AutoPush:       cfg.PushHour >= 0,
		PushHour:       cfg.PushHour,
		PushMinute:     cfg.PushMinute,
		BlacklistCount: blacklistCount,
	}, nil
}

func (s *Service) SetWordCloudEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureWordCloud, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_word_cloud_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) SetWordCloudPushTimeByTGGroupID(tgGroupID int64, raw string) (int, int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, 0, err
	}
	raw = strings.TrimSpace(raw)
	match := wordCloudTimePattern.FindStringSubmatch(raw)
	if len(match) != 3 {
		return 0, 0, errors.New("invalid time format")
	}
	hour, _ := strconv.Atoi(match[1])
	minute, _ := strconv.Atoi(match[2])
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, errors.New("invalid time range")
	}
	cfg, err := s.getWordCloudConfig(group.ID)
	if err != nil {
		return 0, 0, err
	}
	cfg.PushHour = hour
	cfg.PushMinute = minute
	if err := s.saveWordCloudConfig(group.ID, cfg); err != nil {
		return 0, 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_word_cloud_push_time_%02d_%02d", hour, minute), 0, 0)
	return hour, minute, nil
}

func (s *Service) DisableWordCloudAutoPushByTGGroupID(tgGroupID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWordCloudConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.PushHour = -1
	cfg.PushMinute = 0
	if err := s.saveWordCloudConfig(group.ID, cfg); err != nil {
		return err
	}
	_ = s.repo.CreateLog(group.ID, "set_word_cloud_push_time_off", 0, 0)
	return nil
}

func (s *Service) AddWordCloudBlacklistWordByTGGroupID(tgGroupID int64, word string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	word = strings.TrimSpace(strings.ToLower(word))
	if word == "" {
		return errors.New("empty word")
	}
	if err := s.repo.AddWordCloudBlacklistWord(group.ID, word); err != nil {
		return err
	}
	_ = s.repo.CreateLog(group.ID, "word_cloud_black_add", 0, 0)
	return nil
}

func (s *Service) RemoveWordCloudBlacklistWordByTGGroupID(tgGroupID int64, word string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	word = strings.TrimSpace(strings.ToLower(word))
	if word == "" {
		return errors.New("empty word")
	}
	if err := s.repo.RemoveWordCloudBlacklistWord(group.ID, word); err != nil {
		return err
	}
	_ = s.repo.CreateLog(group.ID, "word_cloud_black_remove", 0, 0)
	return nil
}

func (s *Service) ListWordCloudBlacklistByTGGroupID(tgGroupID int64, page, pageSize int) (*WordCloudBlacklistPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListWordCloudBlacklistWordsPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &WordCloudBlacklistPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) collectWordCloudMessage(msg *models.Message, group *model.Group) {
	if msg == nil || group == nil || msg.From == nil || msg.From.IsBot {
		return
	}
	if !s.WordCloudAvailable() {
		return
	}
	if isCommandMessage(msg) {
		return
	}
	content := strings.TrimSpace(antiSpamMessageContent(msg))
	if content == "" {
		return
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureWordCloud, false)
	if err != nil || !enabled {
		return
	}
	user, err := s.repo.UpsertUserFromTG(msg.From)
	if err != nil || user == nil {
		return
	}
	tokenCounts, total, err := s.segmentWordCloudTokens(content)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("word cloud tokenize failed chat=%d user=%d err=%v", msg.Chat.ID, msg.From.ID, err)
		}
		return
	}
	dayKey := wordCloudDayKey(time.Now())
	if err := s.repo.AddWordCloudMessageAndTokens(group.ID, user.ID, dayKey, tokenCounts, total); err != nil && s.logger != nil {
		s.logger.Printf("word cloud persist failed group=%d user=%d err=%v", group.ID, user.ID, err)
	}
}

func (s *Service) segmentWordCloudTokens(content string) (map[string]int, int, error) {
	jieba, err := s.getWordCloudJieba()
	if err != nil {
		return nil, 0, err
	}
	words := jieba.Cut(content, true)

	out := make(map[string]int, len(words))
	total := 0
	for _, raw := range words {
		word, ok := normalizeWordCloudToken(raw)
		if !ok {
			continue
		}
		if _, blocked := wordCloudDefaultStopWords[word]; blocked {
			continue
		}
		out[word]++
		total++
	}
	return out, total, nil
}

func normalizeWordCloudToken(raw string) (string, bool) {
	word := strings.TrimSpace(strings.ToLower(raw))
	if word == "" {
		return "", false
	}
	if utf8.RuneCountInString(word) <= 1 {
		return "", false
	}
	if wordCloudNonWordPattern.MatchString(word) {
		return "", false
	}
	if isWordCloudAllDigit(word) {
		return "", false
	}
	if isWordCloudMostlySymbol(word) {
		return "", false
	}
	return word, true
}

func isWordCloudAllDigit(word string) bool {
	for _, r := range word {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isWordCloudMostlySymbol(word string) bool {
	letters := 0
	for _, r := range word {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters++
		}
	}
	return letters == 0
}

func (s *Service) getWordCloudJieba() (*gojieba.Jieba, error) {
	if s.jieba != nil {
		return s.jieba, nil
	}
	return nil, errors.New("word cloud jieba is not initialized")
}

func (s *Service) WordCloudAvailable() bool {
	return s != nil && s.jieba != nil
}

func (s *Service) preInitWordCloudJieba() error {
	args, err := s.resolveWordCloudJiebaInitArgs()
	if err != nil {
		return err
	}
	jieba, err := newJiebaSafe(args...)
	if err != nil {
		return err
	}
	s.jieba = jieba
	return nil
}

func newJiebaSafe(paths ...string) (_ *gojieba.Jieba, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("init gojieba panic: %v", rec)
		}
	}()
	return gojieba.NewJieba(paths...), nil
}

func (s *Service) resolveWordCloudJiebaInitArgs() ([]string, error) {
	// WORDCLOUD_JIEBA_DICT_DIR 语义：词典目录（自动读取固定 5 个文件名）。
	if dir := strings.TrimSpace(s.wordCloudDict); dir != "" {
		paths := wordCloudJiebaPathsIfValid(dir)
		if paths == nil {
			return nil, fmt.Errorf("invalid WORDCLOUD_JIEBA_DICT_DIR: %s", dir)
		}
		// 对齐 gojieba 默认全局变量语义：DICT_DIR=WORDCLOUD_JIEBA_DICT_DIR
		gojieba.DICT_DIR = dir
		gojieba.DICT_PATH = paths[0]
		gojieba.HMM_PATH = paths[1]
		gojieba.USER_DICT_PATH = paths[2]
		gojieba.IDF_PATH = paths[3]
		gojieba.STOP_WORDS_PATH = paths[4]
		// 使用默认参数，让 getDictPaths 走上面更新后的全局路径。
		return []string{}, nil
	}
	// 未配置目录时，沿用 gojieba 默认路径。
	return []string{}, nil
}

func wordCloudJiebaPathsIfValid(dir string) []string {
	base := strings.TrimSpace(dir)
	if base == "" {
		return nil
	}
	paths := make([]string, 0, len(wordCloudJiebaDictFiles))
	for _, name := range wordCloudJiebaDictFiles {
		p := filepath.Join(base, name)
		if _, err := os.Stat(p); err != nil {
			return nil
		}
		paths = append(paths, p)
	}
	return paths
}

func (s *Service) SendWordCloudReportByTGGroupID(bot *tgbot.Bot, tgGroupID int64, manual bool) error {
	if bot == nil {
		return errors.New("nil bot")
	}
	image, caption, dayKey, err := s.GenerateWordCloudReportByTGGroupID(tgGroupID, time.Now())
	if err != nil {
		return err
	}
	if _, err := bot.SendPhoto(context.Background(), &tgbot.SendPhotoParams{
		ChatID:  tgGroupID,
		Photo:   &models.InputFileUpload{Filename: "wordcloud.png", Data: bytes.NewReader(image)},
		Caption: caption,
	}); err != nil {
		return err
	}
	group, gErr := s.repo.FindGroupByTGID(tgGroupID)
	if gErr == nil {
		action := "word_cloud_send_auto"
		if manual {
			action = "word_cloud_send_manual"
		}
		_ = s.repo.CreateLog(group.ID, action, 0, 0)
		if !manual {
			_ = s.markWordCloudPushed(group.ID, dayKey)
		}
	}
	return nil
}

func isCommandMessage(msg *models.Message) bool {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return false
	}
	for _, e := range msg.Entities {
		if e.Type == models.MessageEntityTypeBotCommand && e.Offset == 0 {
			return true
		}
	}
	return false
}

func (s *Service) GenerateWordCloudReportByTGGroupID(tgGroupID int64, now time.Time) ([]byte, string, string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, "", "", err
	}
	dayKey := wordCloudDayKey(now)
	wordRows, err := s.repo.ListWordCloudWordStats(group.ID, dayKey, wordCloudWordLimit)
	if err != nil {
		return nil, "", "", err
	}
	if len(wordRows) == 0 {
		return nil, "", "", errors.New("今天暂无可用词云数据")
	}
	blacklistRows, err := s.repo.ListWordCloudBlacklistWords(group.ID)
	if err != nil {
		return nil, "", "", err
	}
	blackSet := make(map[string]struct{}, len(blacklistRows))
	for _, row := range blacklistRows {
		blackSet[strings.ToLower(strings.TrimSpace(row.Word))] = struct{}{}
	}
	wordFreq := make(map[string]int, len(wordRows))
	for _, row := range wordRows {
		word := strings.ToLower(strings.TrimSpace(row.Word))
		if word == "" {
			continue
		}
		if _, blocked := blackSet[word]; blocked {
			continue
		}
		if _, blocked := wordCloudDefaultStopWords[word]; blocked {
			continue
		}
		if row.Total <= 0 {
			continue
		}
		wordFreq[word] = int(row.Total)
	}
	if len(wordFreq) == 0 {
		return nil, "", "", errors.New("黑名单过滤后暂无可用词云数据")
	}

	type kv struct {
		Word  string
		Count int
	}
	pairs := make([]kv, 0, len(wordFreq))
	for w, c := range wordFreq {
		pairs = append(pairs, kv{Word: w, Count: c})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Word < pairs[j].Word
		}
		return pairs[i].Count > pairs[j].Count
	})
	if len(pairs) > wordCloudDrawWordLimit {
		pairs = pairs[:wordCloudDrawWordLimit]
	}
	drawMap := make(map[string]int, len(pairs))
	for _, p := range pairs {
		drawMap[p.Word] = p.Count
	}

	fontFile, err := s.resolveWordCloudFontFile()
	if err != nil {
		return nil, "", "", err
	}
	wc := wordclouds.NewWordcloud(drawMap,
		wordclouds.FontFile(fontFile),
		wordclouds.Width(1200),
		wordclouds.Height(680),
		wordclouds.FontMaxSize(150),
		wordclouds.FontMinSize(18),
		wordclouds.BackgroundColor(color.RGBA{R: 252, G: 251, B: 246, A: 255}),
		wordclouds.Colors([]color.Color{
			color.RGBA{R: 41, G: 128, B: 185, A: 255},
			color.RGBA{R: 22, G: 160, B: 133, A: 255},
			color.RGBA{R: 192, G: 57, B: 43, A: 255},
			color.RGBA{R: 142, G: 68, B: 173, A: 255},
			color.RGBA{R: 39, G: 174, B: 96, A: 255},
		}),
	)
	img := wc.Draw()
	var imgBuf bytes.Buffer
	if err := png.Encode(&imgBuf, img); err != nil {
		return nil, "", "", err
	}

	summary, err := s.repo.WordCloudDailySummary(group.ID, dayKey)
	if err != nil {
		return nil, "", "", err
	}
	topRows, err := s.repo.TopWordCloudContributors(group.ID, dayKey, wordCloudContributorSize)
	if err != nil {
		return nil, "", "", err
	}
	localNow := now.In(time.Local)
	dateText := localNow.Format("2006-01-02")
	timeText := localNow.Format("15:04")
	lines := []string{
		fmt.Sprintf("🎤 今日话题词云 %s 🎤", dateText),
		fmt.Sprintf("⏰ 截至今天 %s", timeText),
		fmt.Sprintf("🗣️ 本群 %d 位朋友共产生 %d 条发言", summary.UsersTotal, summary.MessageTotal),
		"🔍 看下有没有你感兴趣的关键词？",
		"",
		"🏵 活跃用户排行榜 🏵",
		"",
	}
	if len(topRows) == 0 {
		lines = append(lines, "今日暂无贡献排行")
	} else {
		for i, row := range topRows {
			rankIcon := "🎖"
			switch i {
			case 0:
				rankIcon = "🥇"
			case 1:
				rankIcon = "🥈"
			case 2:
				rankIcon = "🥉"
			}
			name := fmt.Sprintf("uid:%d", row.UserID)
			if user, uErr := s.repo.FindUserByID(row.UserID); uErr == nil && user != nil {
				if strings.TrimSpace(user.Username) != "" {
					name = user.Username
				} else {
					display := strings.TrimSpace(user.FirstName + " " + user.LastName)
					if display != "" {
						name = display
					}
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s  贡献: %d", rankIcon, name, row.TokenTotal))
		}
	}
	lines = append(lines, "", "🎉感谢这些朋友今天的分享!🎉")
	caption := strings.Join(lines, "\n")
	return imgBuf.Bytes(), caption, dayKey, nil
}

func (s *Service) resolveWordCloudFontFile() (string, error) {
	candidates := make([]string, 0, 8)
	if strings.TrimSpace(s.wordCloudFont) != "" {
		candidates = append(candidates, strings.TrimSpace(s.wordCloudFont))
	}
	candidates = append(candidates,
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/Library/Fonts/Arial Unicode.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	)
	winDir := strings.TrimSpace(os.Getenv("WINDIR"))
	if winDir == "" {
		winDir = `C:\Windows`
	}
	candidates = append(candidates,
		filepath.Join(winDir, "Fonts", "msyh.ttc"),   // Microsoft YaHei
		filepath.Join(winDir, "Fonts", "msyhbd.ttc"), // Microsoft YaHei Bold
		filepath.Join(winDir, "Fonts", "simhei.ttf"), // SimHei
		filepath.Join(winDir, "Fonts", "simsun.ttc"), // SimSun
		filepath.Join(winDir, "Fonts", "simkai.ttf"), // KaiTi
	)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if strings.TrimSpace(s.wordCloudFont) != "" {
		abs, _ := filepath.Abs(s.wordCloudFont)
		return "", fmt.Errorf("词云字体不存在，请检查 WORDCLOUD_FONT_PATH: %s", abs)
	}
	return "", errors.New("未找到可用中文字体，请配置 WORDCLOUD_FONT_PATH")
}

func (s *Service) wordCloudReadyToPush(groupID uint, now time.Time) (bool, error) {
	enabled, err := s.IsFeatureEnabled(groupID, featureWordCloud, false)
	if err != nil || !enabled {
		return false, err
	}
	cfg, err := s.getWordCloudConfig(groupID)
	if err != nil {
		return false, err
	}
	if cfg.PushHour < 0 {
		return false, nil
	}
	current := now.In(time.Local)
	if current.Hour() != cfg.PushHour || current.Minute() != cfg.PushMinute {
		return false, nil
	}
	dayKey := wordCloudDayKey(current)
	if cfg.LastPushDay == dayKey {
		return false, nil
	}
	return true, nil
}

func (s *Service) markWordCloudPushed(groupID uint, dayKey string) error {
	cfg, err := s.getWordCloudConfig(groupID)
	if err != nil {
		return err
	}
	cfg.LastPushDay = strings.TrimSpace(dayKey)
	return s.saveWordCloudConfig(groupID, cfg)
}
