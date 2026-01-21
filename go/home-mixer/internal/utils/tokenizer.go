package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// TweetTokenizer Twitter文本分词器
// 用于将推文文本分解为tokens，识别用户名、标签、URL等特殊元素
type TweetTokenizer struct {
	// 预编译的正则表达式，用于匹配不同类型的tokens
	mentionRegex    *regexp.Regexp // @用户名
	hashtagRegex    *regexp.Regexp // #标签
	urlRegex        *regexp.Regexp // URL
	emoticonRegex   *regexp.Regexp // 表情符号
	wordRegex       *regexp.Regexp // 普通单词
	punctuationRegex *regexp.Regexp // 标点符号
	numberRegex     *regexp.Regexp // 数字
	allTokensRegex  *regexp.Regexp // 组合所有类型的正则
}

// NewTweetTokenizer 创建新的TweetTokenizer实例
func NewTweetTokenizer() *TweetTokenizer {
	// 定义各种token类型的正则表达式
	mentionPattern := `@[\w_]+`              // @username
	hashtagPattern := `#[\w_]+`              // #hashtag
	urlPattern := `https?://\S+`             // http:// or https:// URLs
	emoticonPattern := `[=;][oO\-]?[D\)\]\(\]/\\OpP]` // 表情符号，如 :D, :), :( 等
	wordPattern := `[a-zA-Z][a-zA-Z'\-_]*[a-zA-Z]|[a-zA-Z]` // 单词（可能包含连字符、撇号）
	punctuationPattern := `[!?.,;:]+`        // 标点符号
	numberPattern := `\d+[\d,.]*\d*|\d+`     // 数字（可能包含逗号、小数点）

	// 组合所有模式，按优先级排序（长的模式先匹配）
	// 注意：URL必须在mention和hashtag之前匹配，因为URL可能包含@和#
	allPattern := strings.Join([]string{
		urlPattern,
		emoticonPattern,
		mentionPattern,
		hashtagPattern,
		numberPattern,
		wordPattern,
		punctuationPattern,
		`\S`, // 任何其他非空白字符
	}, "|")

	return &TweetTokenizer{
		mentionRegex:    regexp.MustCompile(mentionPattern),
		hashtagRegex:    regexp.MustCompile(hashtagPattern),
		urlRegex:        regexp.MustCompile(urlPattern),
		emoticonRegex:   regexp.MustCompile(emoticonPattern),
		wordRegex:       regexp.MustCompile(wordPattern),
		punctuationRegex: regexp.MustCompile(punctuationPattern),
		numberRegex:     regexp.MustCompile(numberPattern),
		allTokensRegex:  regexp.MustCompile(allPattern),
	}
}

// Tokenize 将文本分解为tokens
// 返回token序列，保持原始的大小写（除非指定lowercase）
func (tt *TweetTokenizer) Tokenize(text string, lowercase bool) []string {
	if text == "" {
		return []string{}
	}

	// 使用正则表达式查找所有匹配的tokens
	matches := tt.allTokensRegex.FindAllString(text, -1)

	tokens := make([]string, 0, len(matches))
	for _, match := range matches {
		// 对于表情符号，保持原始大小写
		if tt.emoticonRegex.MatchString(match) {
			tokens = append(tokens, match)
		} else if lowercase {
			tokens = append(tokens, strings.ToLower(match))
		} else {
			tokens = append(tokens, match)
		}
	}

	return tokens
}

// TokenSequence 表示一个token序列
// 用于匹配操作
type TokenSequence struct {
	Tokens []string
}

// NewTokenSequence 从token列表创建TokenSequence
func NewTokenSequence(tokens []string) *TokenSequence {
	return &TokenSequence{
		Tokens: tokens,
	}
}

// UserMutes 表示用户的静音关键词token序列列表
type UserMutes struct {
	TokenSequences []*TokenSequence
}

// NewUserMutes 从token序列列表创建UserMutes
func NewUserMutes(tokenSequences []*TokenSequence) *UserMutes {
	return &UserMutes{
		TokenSequences: tokenSequences,
	}
}

// MatchTweetGroup 用于匹配推文是否包含静音关键词的匹配器
type MatchTweetGroup struct {
	userMutes *UserMutes
}

// NewMatchTweetGroup 创建新的MatchTweetGroup
func NewMatchTweetGroup(userMutes *UserMutes) *MatchTweetGroup {
	return &MatchTweetGroup{
		userMutes: userMutes,
	}
}

// Matches 检查token序列是否匹配任何静音关键词
// 使用子序列匹配：如果推文的token序列中包含静音关键词的token序列作为连续子序列，则匹配
func (mtg *MatchTweetGroup) Matches(tweetTokenSequence *TokenSequence) bool {
	if mtg.userMutes == nil || len(mtg.userMutes.TokenSequences) == 0 {
		return false
	}

	tweetTokens := tweetTokenSequence.Tokens

	// 检查是否任何静音关键词序列是推文token序列的子序列
	for _, muteSeq := range mtg.userMutes.TokenSequences {
		if mtg.isSubsequence(muteSeq.Tokens, tweetTokens) {
			return true
		}
	}

	return false
}

// isSubsequence 检查sub是否是seq的连续子序列
// 例如: sub=["the", "cat"] 是 seq=["the", "cat", "sat"] 的子序列
func (mtg *MatchTweetGroup) isSubsequence(sub, seq []string) bool {
	if len(sub) == 0 {
		return true // 空序列是任何序列的子序列
	}
	if len(sub) > len(seq) {
		return false
	}

	// 使用滑动窗口查找连续子序列
	for i := 0; i <= len(seq)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if !mtg.tokensMatch(sub[j], seq[i+j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// tokensMatch 检查两个token是否匹配
// 这里使用精确匹配（区分大小写），也可以改为忽略大小写
func (mtg *MatchTweetGroup) tokensMatch(token1, token2 string) bool {
	// 精确匹配
	return token1 == token2
}

// TokenizeWords 将文本按单词边界分解（简单的单词tokenization）
// 用于更简单的匹配场景
func TokenizeWords(text string, lowercase bool) []string {
	// 移除标点符号，保留字母、数字、连字符
	text = strings.TrimSpace(text)
	
	tokens := []string{}
	current := strings.Builder{}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '\'' {
			current.WriteRune(unicode.ToLower(r))
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	if !lowercase {
		// 如果需要保持原始大小写，需要重新处理
		return strings.FieldsFunc(text, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '\''
		})
	}

	return tokens
}
