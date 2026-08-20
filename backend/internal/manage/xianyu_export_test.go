package manage

import (
	"strings"
	"testing"

	"yingyan.local/backend/internal/redemption"
)

func TestBuildXianyuInventoryUsesOneEncodedClaimPerLine(t *testing.T) {
	codes := []redemption.GeneratedCode{
		{FullCode: "YY-AAAA-BBBB-CCCC"},
		{FullCode: "YY-A B+C-EEEE-FFFF"},
	}
	content := buildXianyuInventory("https://img.example.com/", codes)
	lines := strings.Split(content, "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	if !strings.Contains(lines[0], "浏览器打开领取并开始创作吧～：https://img.example.com/claim?code=YY-AAAA-BBBB-CCCC") ||
		!strings.HasSuffix(lines[0], "兑换码：YY-AAAA-BBBB-CCCC") {
		t.Fatalf("first line = %q", lines[0])
	}
	if !strings.Contains(lines[1], "code=YY-A+B%2BC-EEEE-FFFF") ||
		!strings.HasSuffix(lines[1], "兑换码：YY-A B+C-EEEE-FFFF") {
		t.Fatalf("encoded line = %q", lines[1])
	}
}
