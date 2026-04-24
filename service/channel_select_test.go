package service

import "testing"

func TestShouldShowUnavailableTokenGroupMessageOnlyForMissingGroup(t *testing.T) {
	if !ShouldShowUnavailableTokenGroupMessage("missing_group_for_test", "gpt-image-2") {
		t.Fatal("expected missing group to show unavailable token group message")
	}

	if ShouldShowUnavailableTokenGroupMessage("default", "gpt-image-2") {
		t.Fatal("expected existing group without model channel to use no-available-channel message")
	}
}
