package handler

import (
	"reflect"
	"testing"
)

func TestParseWordCloudBlacklistBatch(t *testing.T) {
	got := parseWordCloudBlacklistBatch("  btc \neth\n\nbtc\r\nsol")
	want := []string{"btc", "eth", "sol"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
