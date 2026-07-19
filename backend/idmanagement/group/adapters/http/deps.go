package http

import httpdeps "github.com/ambi/idmagic/backend/idmanagement/adapters/http/httpdeps"

// Deps is an alias for httpdeps.Deps so this package's handler signatures
// can keep referring to the plain "Deps" name (ADR-130 Phase 2).
type Deps = httpdeps.Deps
