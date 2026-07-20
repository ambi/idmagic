package bootstrap

import (
	"context"
	"fmt"
	"strings"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	breachesHIBP "github.com/ambi/idmagic/backend/shared/policy/breaches_hibp"
	breachesNoop "github.com/ambi/idmagic/backend/shared/policy/breaches_noop"
)

// breachedPasswordCheckerVersion は HIBP の User-Agent に乗せる版番号 (HIBP の etiquette)。
const breachedPasswordCheckerVersion = "0.3.0"

// resolveBreachedPasswordChecker は BREACHED_PASSWORD_CHECKER 環境変数から
// BreachedPasswordChecker adapter を組み立てる。既定は noop (外部依存なし)。
// hibp 選択時は api.pwnedpasswords.com への egress が要る (ADR-028 §3)。
func ResolveBreachedPasswordChecker(getenv func(string) string) (passwordports.BreachedPasswordChecker, error) {
	kind := strings.ToLower(strings.TrimSpace(getenv("BREACHED_PASSWORD_CHECKER")))
	if kind == "" {
		kind = "noop"
	}
	switch kind {
	case "noop":
		logging.Info(context.Background(), "breached password checker configured", "kind", "noop")
		return breachesNoop.NoopBreachedPasswordChecker{}, nil
	case "hibp":
		logging.Info(context.Background(), "breached password checker configured", "kind", "hibp")
		return breachesHIBP.NewHibpBreachedPasswordChecker(breachedPasswordCheckerVersion), nil
	default:
		return nil, fmt.Errorf("unsupported BREACHED_PASSWORD_CHECKER=%q (want noop or hibp)", kind)
	}
}
