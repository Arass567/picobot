package whatsapp

import "testing"

func TestParseYesIntent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "oui", input: "OUI", want: true},
		{name: "yes", input: "yes", want: true},
		{name: "spaces", input: "  oui  ", want: true},
		{name: "other", input: "non", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseYesIntent(tt.input); got != tt.want {
				t.Fatalf("ParseYesIntent(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
