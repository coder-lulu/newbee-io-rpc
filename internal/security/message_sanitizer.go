package security

import (
	"fmt"
	"regexp"
)

// MessageSanitizer 消息脱敏器
type MessageSanitizer struct {
	sensitivePatterns []*regexp.Regexp
}

// NewMessageSanitizer 创建消息脱敏器
func NewMessageSanitizer() *MessageSanitizer {
	return &MessageSanitizer{
		sensitivePatterns: []*regexp.Regexp{
			// 匹配常见的敏感字段
			regexp.MustCompile(`(?i)"password"\s*:\s*"[^"]+"`),
			regexp.MustCompile(`(?i)"api_key"\s*:\s*"[^"]+"`),
			regexp.MustCompile(`(?i)"secret"\s*:\s*"[^"]+"`),
			regexp.MustCompile(`(?i)"token"\s*:\s*"[^"]+"`),
			regexp.MustCompile(`(?i)"private_key"\s*:\s*"[^"]+"`),
			regexp.MustCompile(`(?i)"access_key"\s*:\s*"[^"]+"`),
			regexp.MustCompile(`(?i)"secret_key"\s*:\s*"[^"]+"`),
		},
	}
}

// Sanitize 脱敏消息，返回脱敏后的消息和警告列表
func (s *MessageSanitizer) Sanitize(body []byte) ([]byte, []string) {
	var warnings []string
	sanitized := body

	for _, pattern := range s.sensitivePatterns {
		if pattern.Match(body) {
			warnings = append(warnings, fmt.Sprintf("Detected sensitive pattern: %s", pattern.String()))
			// 替换为占位符
			sanitized = pattern.ReplaceAll(sanitized, []byte(`"***REDACTED***"`))
		}
	}

	return sanitized, warnings
}

// Validate 验证消息是否包含敏感信息，如果包含则返回错误
func (s *MessageSanitizer) Validate(body []byte) error {
	for _, pattern := range s.sensitivePatterns {
		if pattern.Match(body) {
			return fmt.Errorf("message contains sensitive field: %s", pattern.String())
		}
	}
	return nil
}
