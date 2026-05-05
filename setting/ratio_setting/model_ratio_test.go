package ratio_setting

import "testing"

func resetModelBillingMapsForTest(t *testing.T) {
	t.Helper()

	originalPriceMap := modelPriceMap.ReadAll()
	originalRatioMap := modelRatioMap.ReadAll()

	modelPriceMap.Clear()
	modelRatioMap.Clear()

	t.Cleanup(func() {
		modelPriceMap.Clear()
		modelRatioMap.Clear()
		modelPriceMap.AddAll(originalPriceMap)
		modelRatioMap.AddAll(originalRatioMap)
	})
}

func TestGetCompletionRatioInfoGPT55UsesOfficialOutputMultiplier(t *testing.T) {
	info := GetCompletionRatioInfo("gpt-5.5")

	if info.Ratio != 6 {
		t.Fatalf("gpt-5.5 completion ratio = %v, want 6", info.Ratio)
	}
	if !info.Locked {
		t.Fatal("gpt-5.5 completion ratio should be locked to the official multiplier")
	}
}

func TestGetCompletionRatioGPT55DatedVariant(t *testing.T) {
	got := GetCompletionRatio("gpt-5.5-2026-04-24")

	if got != 6 {
		t.Fatalf("gpt-5.5 dated variant completion ratio = %v, want 6", got)
	}
}

func TestGetModelPriceFallsBackToBaseModelForCompact(t *testing.T) {
	resetModelBillingMapsForTest(t)

	modelPriceMap.Set("gpt-5.5", 12.34)

	got, ok := GetModelPrice("gpt-5.5-openai-compact", false)
	if !ok {
		t.Fatal("expected compact model price to fall back to base model price")
	}
	if got != 12.34 {
		t.Fatalf("compact model price = %v, want 12.34", got)
	}
}

func TestGetModelPricePrefersCompactWildcardOverBaseModel(t *testing.T) {
	resetModelBillingMapsForTest(t)

	modelPriceMap.Set("gpt-5.5", 12.34)
	modelPriceMap.Set(CompactWildcardModelKey, 5.67)

	got, ok := GetModelPrice("gpt-5.5-openai-compact", false)
	if !ok {
		t.Fatal("expected compact model price to use compact wildcard price")
	}
	if got != 5.67 {
		t.Fatalf("compact wildcard price = %v, want 5.67", got)
	}
}

func TestGetModelRatioFallsBackToBaseModelForCompact(t *testing.T) {
	resetModelBillingMapsForTest(t)

	modelRatioMap.Set("gpt-5.5", 4.56)

	got, ok, matchName := GetModelRatio("gpt-5.5-openai-compact")
	if !ok {
		t.Fatal("expected compact model ratio to fall back to base model ratio")
	}
	if got != 4.56 {
		t.Fatalf("compact model ratio = %v, want 4.56", got)
	}
	if matchName != "gpt-5.5" {
		t.Fatalf("compact model matched name = %q, want %q", matchName, "gpt-5.5")
	}
}
