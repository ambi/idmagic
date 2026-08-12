package bootstrap

import (
	"context"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	breachesHIBP "github.com/ambi/idmagic/backend/shared/policy/breaches_hibp"
	breachesNoop "github.com/ambi/idmagic/backend/shared/policy/breaches_noop"
)

// breachedPasswordCheckerVersion は HIBP の User-Agent に乗せる版番号 (HIBP の etiquette)。
const breachedPasswordCheckerVersion = "0.3.0"

// ResolveBreachedPasswordChecker builds the BreachedPasswordChecker adapter
// selected by cfg.BreachedPasswordChecker. cfg is assumed already validated
// by LoadSharedConfig (BREACHED_PASSWORD_CHECKER is a closed enum), so this
// function only constructs the adapter. hibp 選択時は api.pwnedpasswords.com
// への egress が要る。
func ResolveBreachedPasswordChecker(cfg SharedConfig) passwordports.BreachedPasswordChecker {
	if cfg.BreachedPasswordChecker == "hibp" {
		logging.Info(context.Background(), "breached password checker configured", "kind", "hibp")
		return breachesHIBP.NewHibpBreachedPasswordChecker(breachedPasswordCheckerVersion)
	}
	logging.Info(context.Background(), "breached password checker configured", "kind", "noop")
	return breachesNoop.NoopBreachedPasswordChecker{}
}
