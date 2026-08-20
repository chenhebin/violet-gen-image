package manage

import (
	"fmt"
	"net/url"
	"strings"

	"yingyan.local/backend/internal/redemption"
)

func buildXianyuInventory(
	publicWebURL string,
	codes []redemption.GeneratedCode,
) string {
	var builder strings.Builder
	for index, code := range codes {
		query := url.Values{"code": []string{code.FullCode}}
		claimURL := strings.TrimRight(publicWebURL, "/") + "/claim?" + query.Encode()
		if index > 0 {
			builder.WriteByte('\n')
		}
		fmt.Fprintf(
			&builder,
			"浏览器打开领取并开始创作吧～：%s ｜ 兑换码：%s",
			claimURL,
			code.FullCode,
		)
	}
	return builder.String()
}
