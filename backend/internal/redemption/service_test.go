package redemption

import (
	"testing"
	"time"

	"yingyan.local/backend/internal/model"
)

func TestStatusPriority(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	redeemed := now.Add(-2 * time.Hour)
	disabled := now.Add(-3 * time.Hour)

	tests := []struct {
		name string
		code model.RedemptionCode
		want string
	}{
		{name: "unused", code: model.RedemptionCode{}, want: StatusUnused},
		{name: "expired", code: model.RedemptionCode{ExpiresAt: &past}, want: StatusExpired},
		{
			name: "disabled before expired",
			code: model.RedemptionCode{ExpiresAt: &past, DisabledAt: &disabled},
			want: StatusDisabled,
		},
		{
			name: "redeemed before every other state",
			code: model.RedemptionCode{
				ExpiresAt: &past, DisabledAt: &disabled, RedeemedAt: &redeemed,
			},
			want: StatusRedeemed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Status(test.code, now); got != test.want {
				t.Fatalf("Status() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGeneratedCodeShape(t *testing.T) {
	value, err := randomCharacters(4)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 4 {
		t.Fatalf("length = %d, want 4", len(value))
	}
}
