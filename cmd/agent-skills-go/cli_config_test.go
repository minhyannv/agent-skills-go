package main

import "testing"

func TestStringSliceFlagSetRejectsComma(t *testing.T) {
	var f stringSliceFlag
	if err := f.Set("./skills,../shared-skills"); err == nil {
		t.Fatal("expected comma-separated value to be rejected")
	}
}

func TestStringSliceFlagSetAcceptsSingleValue(t *testing.T) {
	var f stringSliceFlag
	if err := f.Set("./skills"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f) != 1 || f[0] != "./skills" {
		t.Fatalf("unexpected flag values: %#v", f)
	}
}
