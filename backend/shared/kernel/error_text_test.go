package kernel_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/kernel"
)

func TestEnglishErrorText(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		text string
		want string
	}{
		{name: "English is preserved", text: "Invalid request.", want: "Invalid request."},
		{name: "Japanese is replaced", text: "リクエストが不正です", want: "The request could not be completed."},
		{name: "Mixed text is replaced", text: "invalid request が発生しました", want: "The request could not be completed."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := kernel.EnglishErrorText(tt.text); got != tt.want {
				t.Errorf("EnglishErrorText(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
