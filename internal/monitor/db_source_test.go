package monitor

import (
	"strings"
	"testing"
)

func TestLogScopeCondExcludesGatewayBlockedErrors(t *testing.T) {
	cond := logScopeCond()

	if !strings.Contains(cond, `"prompt_check"`) {
		t.Fatal("统计范围应排除 prompt 检查拦截产生的错误日志")
	}
	if !strings.Contains(cond, `"leak_protection_reason"`) {
		t.Fatal("统计范围应排除泄漏保护拦截产生的错误日志")
	}
	if !strings.Contains(cond, "COALESCE(other, '')") {
		t.Fatal("other 可能为 NULL，必须用 COALESCE 兜底，否则普通错误日志会被 NOT LIKE 一并排除")
	}
	if got := strings.Count(cond, "?"); got != 2 {
		t.Fatalf("条件应恰好保留 2 个占位符（消费类型、错误类型），实际 %d 个", got)
	}
}
