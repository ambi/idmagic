package sourcingdeps

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
)

//spec:covers REQ-SOURCING-016: IdManagement の拒否は、項目の失敗として読む port のエラーへ写り、元のエラーも保つ。ほかのエラーは写さない。
func TestRefusalMapsIdManagementRejections(t *testing.T) {
	for source, want := range map[error]error{
		idmanagement.ErrSourceUsernameConflict: directoryports.ErrUsernameConflict,
		idmanagement.ErrSourceEmailConflict:    directoryports.ErrEmailConflict,
		idmanagement.ErrSourceInvalidAttribute: directoryports.ErrInvalidAttribute,
	} {
		got := refusal(fmt.Errorf("wrapped: %w", source))
		if !errors.Is(got, want) || !errors.Is(got, source) {
			t.Errorf("refusal(%v) = %v, want %v", source, got, want)
		}
	}
	other := errors.New("database unavailable")
	got := refusal(other)
	for _, refused := range []error{directoryports.ErrUsernameConflict, directoryports.ErrEmailConflict, directoryports.ErrInvalidAttribute} {
		if errors.Is(got, refused) {
			t.Errorf("refusal(other) = %v, mapped to %v", got, refused)
		}
	}
	if !errors.Is(got, other) {
		t.Errorf("refusal(other) = %v, lost the original error", got)
	}
	if refusal(nil) != nil {
		t.Error("refusal(nil) is not nil")
	}
}
