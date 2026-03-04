package handler

import (
	"reflect"
	"testing"
)

func TestParseBannedWordsBatch(t *testing.T) {
	got := parseBannedWordsBatch("  他妈的 \n尼玛\n\n他妈的\r\n  ")
	want := []string{"他妈的", "尼玛"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestParseBannedWordsBatchEmpty(t *testing.T) {
	got := parseBannedWordsBatch(" \n\t\r\n ")
	if len(got) != 0 {
		t.Fatalf("want empty result, got %v", got)
	}
}
