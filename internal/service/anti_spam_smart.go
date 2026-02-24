package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var smartLeadPattern = regexp.MustCompile(`(?i)(私聊我|联系我|加我|飞机号|tg号|telegram|whatsapp|vx|v信|微信|line|客服|推广|带单|充值|返现)`)
var smartPhonePattern = regexp.MustCompile(`(?i)(?:\+?86[-\s]?)?1[3-9]\d{9}`)

var smartShortLinkDomains = []string{
	"bit.ly/", "t.cn/", "tinyurl.com/", "goo.gl/", "ow.ly/", "is.gd/", "cutt.ly/", "rebrand.ly/",
	"buff.ly/", "shorturl.at/", "surl.li/", "t.ly/", "rb.gy/", "v.gd/", "clck.ru/", "url.cn/",
	"j.mp/", "u.nu/", "x.co/", "adf.ly/", "soo.gd/", "lnk.bio/", "qrco.de/", "shorte.st/",
	"tiny.one/", "y2u.be/",
}

var smartSpamKeywords = []string{
	"免费", "福利", "兼职", "返利", "推广", "联系", "私聊", "加我", "群发", "资源",
	"客服", "代理", "稳定", "靠谱", "日结", "秒结", "引流", "精准粉", "拉群", "代发",
	"合作", "咨询", "项目", "赚米", "爆粉", "挂机", "副业", "零门槛", "可兼职", "高收益",
	"低投入", "保底", "稳赚", "包赔", "下分", "上分", "彩票", "博彩", "网赚", "返现",
	"佣金", "首充", "充值", "提现", "邀请码", "注册码", "返佣", "拉新", "开户", "交易所",
	"空投", "撸", "usdt", "合约", "带单", "币圈", "钱包", "秒到", "永久", "解封",
	"telegram", "whatsapp", "line", "vx", "v信", "飞机号", "tg号", "私信", "私聊我", "联系我",
	"官方客服", "保证金", "海外盘", "刷单", "出码", "码商", "灰产", "代收", "代付",
}

var smartPornKeywords = []string{
	"约炮", "裸聊", "成人视频", "小姐", "包夜", "上门", "外围", "援交", "兼职女", "同城约",
	"做爱", "一夜情", "车震", "性服务", "激情", "大保健", "特殊服务", "全套", "开房", "过夜",
	"av", "无码", "有码", "国产自拍", "偷拍", "色情", "成人", "黄网", "黄图", "成人直播",
	"女神陪聊", "男技师", "会所", "学生妹", "萝莉", "御姐", "口交", "高潮", "调教", "制服诱惑",
	"情趣", "自慰", "床照", "原味", "乳交", "SM", "做0", "做1", "双飞", "约吗",
	"看片", "porn", "sex", "nude", "escort", "成人视频", "裸舞", "同城艳遇", "性爱", "无码视频",
}

type antiSpamDecision struct {
	Blocked     bool
	Score       int
	Reasons     []string
	Category    string
	ReasonCode  string
	ReasonLabel string
	Smart       bool
}

type smartScoreState struct {
	total   int
	spam    int
	porn    int
	reasons []string
}

func antiSpamDecisionFromRules(msg *tgbotapi.Message, cfg antiSpamConfig) antiSpamDecision {
	blocked, reasonCode, reasonLabel := antiSpamViolation(msg, cfg)
	if !blocked {
		return antiSpamDecision{Blocked: false, Category: "normal"}
	}
	return antiSpamDecision{
		Blocked:     true,
		Score:       100,
		Reasons:     []string{reasonLabel},
		Category:    "spam",
		ReasonCode:  reasonCode,
		ReasonLabel: reasonLabel,
		Smart:       false,
	}
}

func (s *Service) evaluateSmartAntiSpam(tgGroupID int64, msg *tgbotapi.Message, content string, cfg antiSpamConfig) antiSpamDecision {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return antiSpamDecision{Blocked: false, Category: "normal", Smart: true}
	}

	lower := strings.ToLower(trimmed)
	mentions := mentionPattern.FindAllString(trimmed, -1)
	hasLink := containsLink(trimmed)
	hasTMe := strings.Contains(lower, "t.me/") || strings.Contains(lower, "telegram.me/")
	hasShortLink := containsShortLinkDomain(lower)
	hasLead := smartLeadPattern.MatchString(lower)
	hasPhone := smartPhonePattern.MatchString(trimmed)
	hasMedia := isNightMediaMessage(msg)

	score := smartScoreState{}
	if hasLink {
		score.addSpam(35, "外链")
	}
	if hasTMe {
		score.addSpam(30, "t.me 引流")
	}
	if hasShortLink {
		score.addSpam(25, "短链")
	}
	if len(mentions) >= 2 {
		score.addSpam(20, "多个@用户名")
	}
	if hasLead {
		score.addSpam(15, "引流句式")
	}
	if hasPhone {
		score.addSpam(12, "手机号")
	}

	spamHitCount := countKeywordHits(lower, smartSpamKeywords)
	if spamHitCount > 0 {
		points := spamHitCount * 8
		if points > 40 {
			points = 40
		}
		score.addSpam(points, fmt.Sprintf("垃圾词命中%d", spamHitCount))
	}

	pornHitCount := countKeywordHits(lower, smartPornKeywords)
	if pornHitCount > 0 {
		points := pornHitCount * 15
		if points > 60 {
			points = 60
		}
		score.addPorn(points, fmt.Sprintf("色情词命中%d", pornHitCount))
	}

	plain := strings.TrimSpace(mentionPattern.ReplaceAllString(urlPattern.ReplaceAllString(trimmed, ""), ""))
	if utf8.RuneCountInString(plain) < 8 && (hasLink || len(mentions) > 0) {
		score.addSpam(15, "短文本引流")
	}

	if hasMedia && (hasLink || len(mentions) > 0 || hasLead || hasShortLink) {
		score.addSpam(20, "媒体+引流文案")
	}

	if hasLongRepeatedRune(trimmed, 5) || hasRepeatedChunk(strings.Join(strings.Fields(lower), ""), 1, 4, 3) {
		score.addSpam(15, "重复字符/句式")
	}

	if looksAllCaps(trimmed) {
		score.addSpam(10, "英文全大写")
	}

	if isLowInfoText(trimmed) {
		score.addSpam(10, "低信息文本")
	}

	if msg.From != nil {
		norm := normalizeSpamText(trimmed)
		if norm != "" {
			dupCount := s.recordRecentSameMessage(tgGroupID, msg.From.ID, norm)
			if dupCount >= 3 {
				score.addSpam(20, "短时重复刷屏")
			}
		}
		if hasLink || hasTMe || hasShortLink {
			if joinAt, ok := s.getJoinAt(tgGroupID, msg.From.ID); ok && time.Since(joinAt) <= 10*time.Minute {
				score.addSpam(25, "新号短时发链")
			}
		}
	}

	if score.total > 100 {
		score.total = 100
	}
	category := "normal"
	if score.total > 0 {
		category = "spam"
		if score.porn > 0 && score.porn >= score.spam {
			category = "porn"
		}
	}

	decision := antiSpamDecision{
		Blocked:  score.total >= cfg.SmartDeleteScore,
		Score:    score.total,
		Reasons:  score.reasons,
		Category: category,
		Smart:    true,
	}
	if decision.Blocked {
		decision.ReasonCode = fmt.Sprintf("smart_%s_%d", category, score.total)
		decision.ReasonLabel = fmt.Sprintf("智能识别:%s score=%d", category, score.total)
	}
	return decision
}

func (s *Service) recordRecentSameMessage(tgGroupID, tgUserID int64, text string) int {
	if tgUserID == 0 || strings.TrimSpace(text) == "" {
		return 0
	}
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.spamRecent[key]
	valid := make([]floodEvent, 0, len(items)+1)
	count := 1
	for _, item := range items {
		if now-item.Timestamp > 120 {
			continue
		}
		valid = append(valid, item)
		if item.Text == text {
			count++
		}
	}
	valid = append(valid, floodEvent{Timestamp: now, Text: text})
	s.spamRecent[key] = valid
	return count
}

func antiSpamUpgradePenalty(current string) string {
	switch current {
	case antiFloodPenaltyDeleteOnly, antiFloodPenaltyWarn:
		return antiFloodPenaltyMute
	default:
		return current
	}
}

func antiSpamReasonListPreview(reasons []string) string {
	if len(reasons) == 0 {
		return "规则命中"
	}
	if len(reasons) <= 3 {
		return strings.Join(reasons, "、")
	}
	return strings.Join(reasons[:3], "、") + "..."
}

func (s *smartScoreState) addSpam(points int, reason string) {
	if points <= 0 {
		return
	}
	s.total += points
	s.spam += points
	s.reasons = append(s.reasons, reason)
}

func (s *smartScoreState) addPorn(points int, reason string) {
	if points <= 0 {
		return
	}
	s.total += points
	s.porn += points
	s.reasons = append(s.reasons, reason)
}

func countKeywordHits(content string, keywords []string) int {
	if content == "" || len(keywords) == 0 {
		return 0
	}
	hits := 0
	for _, kw := range keywords {
		k := strings.TrimSpace(strings.ToLower(kw))
		if k == "" {
			continue
		}
		if strings.Contains(content, k) {
			hits++
		}
	}
	return hits
}

func containsShortLinkDomain(content string) bool {
	for _, domain := range smartShortLinkDomains {
		if strings.Contains(content, domain) {
			return true
		}
	}
	return false
}

func looksAllCaps(content string) bool {
	letters := 0
	upper := 0
	for _, r := range content {
		if r > unicode.MaxASCII {
			continue
		}
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.IsUpper(r) {
			upper++
		}
	}
	return letters >= 8 && upper*100/letters >= 85
}

func isLowInfoText(content string) bool {
	trimmed := strings.TrimSpace(content)
	if utf8.RuneCountInString(trimmed) < 10 {
		return false
	}
	total := 0
	nonInfo := 0
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			nonInfo++
		}
	}
	if total == 0 {
		return false
	}
	return nonInfo*100/total >= 60
}

func hasLongRepeatedRune(content string, minRun int) bool {
	if minRun <= 1 {
		return strings.TrimSpace(content) != ""
	}
	var (
		prev    rune
		run     int
		hasPrev bool
	)
	for _, r := range content {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			hasPrev = false
			run = 0
			continue
		}
		if !hasPrev || r != prev {
			prev = r
			run = 1
			hasPrev = true
			continue
		}
		run++
		if run >= minRun {
			return true
		}
	}
	return false
}

func hasRepeatedChunk(content string, minChunk, maxChunk, minRepeat int) bool {
	if minRepeat <= 1 || content == "" {
		return false
	}
	runes := []rune(content)
	for size := minChunk; size <= maxChunk; size++ {
		if size <= 0 || len(runes) < size*minRepeat {
			continue
		}
		for start := 0; start+size*minRepeat <= len(runes); start++ {
			chunk := string(runes[start : start+size])
			repeat := 1
			for pos := start + size; pos+size <= len(runes); pos += size {
				if string(runes[pos:pos+size]) != chunk {
					break
				}
				repeat++
				if repeat >= minRepeat {
					return true
				}
			}
		}
	}
	return false
}
