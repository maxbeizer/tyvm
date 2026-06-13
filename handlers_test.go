package main

import "testing"

func TestParseOptionalFloat(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantNil bool
		wantVal float64
		wantErr bool
	}{
		{"empty returns nil no error", "", true, 0, false},
		{"valid float", "7.2", false, 7.2, false},
		{"valid integer", "42", false, 42, false},
		{"invalid returns error", "abc", true, 0, true},
		{"trailing garbage returns error", "7.2x", true, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptionalFloat(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %v, got nil", tc.wantVal)
			}
			if *got != tc.wantVal {
				t.Errorf("expected %v, got %v", tc.wantVal, *got)
			}
		})
	}
}
