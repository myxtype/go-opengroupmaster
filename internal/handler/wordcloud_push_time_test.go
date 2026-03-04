package handler

import "testing"

func TestIsWordCloudAutoPushDisableInput(t *testing.T) {
	cases := []string{"关闭", "不自动推送", "off", " OFF "}
	for _, c := range cases {
		if !isWordCloudAutoPushDisableInput(c) {
			t.Fatalf("expected disable input for %q", c)
		}
	}
	if isWordCloudAutoPushDisableInput("18:00") {
		t.Fatalf("unexpected disable input for time")
	}
}
