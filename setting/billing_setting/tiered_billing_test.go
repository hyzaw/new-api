package billing_setting

import "testing"

func resetBillingSettingForTest(t *testing.T) {
	t.Helper()

	originalMode := billingSetting.BillingMode
	originalExpr := billingSetting.BillingExpr

	billingSetting.BillingMode = make(map[string]string)
	billingSetting.BillingExpr = make(map[string]string)

	t.Cleanup(func() {
		billingSetting.BillingMode = originalMode
		billingSetting.BillingExpr = originalExpr
	})
}

func TestGetBillingModeFallsBackToBaseModelForCompact(t *testing.T) {
	resetBillingSettingForTest(t)

	billingSetting.BillingMode["gpt-5.5"] = BillingModeTieredExpr

	got := GetBillingMode("gpt-5.5-openai-compact")
	if got != BillingModeTieredExpr {
		t.Fatalf("billing mode = %q, want %q", got, BillingModeTieredExpr)
	}
}

func TestGetBillingExprFallsBackToBaseModelForCompact(t *testing.T) {
	resetBillingSettingForTest(t)

	billingSetting.BillingExpr["gpt-5.5"] = `tier("base", p * 2)`

	got, ok := GetBillingExpr("gpt-5.5-openai-compact")
	if !ok {
		t.Fatal("expected compact model to inherit base model billing expr")
	}
	if got != `tier("base", p * 2)` {
		t.Fatalf("billing expr = %q, want base model expr", got)
	}
}
