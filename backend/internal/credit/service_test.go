package credit

import "testing"

func TestLedgerConstantsAreStable(t *testing.T) {
	got := []string{
		LedgerRedemption,
		LedgerReserve,
		LedgerRelease,
		LedgerRefund,
		LedgerAdjustment,
	}
	want := []string{"redemption", "reserve", "release", "refund", "adjustment"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ledger type %d = %q, want %q", index, got[index], want[index])
		}
	}
}
