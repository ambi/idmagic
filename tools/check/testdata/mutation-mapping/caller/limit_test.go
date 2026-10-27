package caller

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/tools/check/testdata/mutation-mapping/target"
)

func TestAtLimit(t *testing.T) {
	if os.Getenv("MUTATION_MAPPING_FAIL_BASELINE") == "1" {
		t.Fatal("baseline failure requested by the mutation runner canary")
	}
	if !target.WithinLimit(2, 2) {
		t.Fatal("value equal to maximum was refused")
	}
}
