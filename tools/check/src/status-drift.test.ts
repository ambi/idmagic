import { describe, expect, it } from 'bun:test'
import {
  collectResponders,
  diffStatusCodes,
  type OpenAPIDocument,
  type StatusDriftResult,
} from './status-drift.ts'

/**
 * What these tests hold is the reading rule, not a list of operations.
 *
 * The reading rule is the whole design: a guard's statuses are all reachable
 * where it is called, an error mapper's are not, and an operation whose writers
 * were not all read must not be reported as over-declaring. Each of those is
 * asserted on the shape it actually takes in this repository.
 */

const responders = (source: string) => collectResponders([{ path: 'h.go', source }])

describe('collectResponders', () => {
  it('reads the constant status of an echo write', () => {
    const found = responders(`
func (d Deps) handleGet(c *echo.Context) error {
	return c.JSON(http.StatusOK, res)
}`)
    expect([...(found.get('handleGet')?.statuses ?? [])]).toEqual([200])
  })

  it('reads the status a shared writer is handed at the call site', () => {
    // WriteProblem takes the status as an argument, so the caller settles it and
    // the callee never has to be read.
    const found = responders(`
func HandleGetAdminUser(d Deps, c *echo.Context) error {
	if user == nil {
		return support.WriteProblem(c, http.StatusNotFound, "user_not_found", "The user does not exist.")
	}
	return support.NoStoreJSON(c, http.StatusOK, res)
}`)
    expect([...(found.get('HandleGetAdminUser')?.statuses ?? [])].sort()).toEqual([200, 404])
  })

  it('ignores a function that cannot reach the response', () => {
    // Only a holder of the context can write one. A repository that compares an
    // upstream response against http.StatusNotFound is not writing 404.
    const found = responders(`
func (r Repo) FindBySub(ctx context.Context, sub string) (*User, error) {
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return user, nil
}`)
    expect(found.has('FindBySub')).toBe(false)
  })

  it('cuts the body at the matching brace so a neighbour does not leak in', () => {
    const found = responders(`
func (d Deps) handleA(c *echo.Context) error {
	if x {
		return c.NoContent(http.StatusNoContent)
	}
	return nil
}

func (d Deps) handleB(c *echo.Context) error {
	return c.JSON(http.StatusConflict, res)
}`)
    expect([...(found.get('handleA')?.statuses ?? [])]).toEqual([204])
    expect([...(found.get('handleB')?.statuses ?? [])]).toEqual([409])
  })

  it('reads a function whose return type carries braces of its own', () => {
    // `map[string]any {` and `struct{}` both put a brace in the signature; the
    // body brace is the one that is neither.
    const found = responders(`
func (d Deps) handleMeta(c *echo.Context) (map[string]any, error) {
	return c.JSON(http.StatusOK, nil), nil
}`)
    expect([...(found.get('handleMeta')?.statuses ?? [])]).toEqual([200])
  })
})

describe('guards and error mappers', () => {
  it('follows a guard, because every branch of it is reachable where it is called', () => {
    // VerifyBrowserRequest decides from the request in front of it, so an
    // operation that calls it can answer 403 whatever its use case does.
    const found = responders(`
func (d Deps) VerifyBrowserRequest(c *echo.Context) error {
	if origin != want {
		_ = WriteProblem(c, http.StatusForbidden, "invalid_origin", "The request origin does not match.")
		return ErrBrowserVerificationFailed
	}
	return nil
}

func (d Deps) handleCreate(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, res)
}`)
    expect([...(found.get('handleCreate')?.statuses ?? [])].sort()).toEqual([201, 403])
    expect(found.get('handleCreate')?.unread).toEqual([])
  })

  it('follows a guard that only delegates to another guard', () => {
    const found = responders(`
func (d Deps) WriteAdminAccessError(c *echo.Context, err error) error {
	if errors.Is(err, ErrAdminAuthenticationRequired) {
		return WriteProblem(c, http.StatusUnauthorized, "authentication_required", "x")
	}
	return WriteProblem(c, http.StatusForbidden, "access_denied", "y")
}

func (d Deps) requireAdmin(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return nil
}

func (d Deps) handleList(c *echo.Context) error {
	if err := d.requireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`)
    expect([...(found.get('handleList')?.statuses ?? [])].sort()).toEqual([200, 401, 403])
    expect(found.get('handleList')?.unread).toEqual([])
  })

  it('refuses to follow a helper that maps an error value, and says so', () => {
    // WriteAccountError answers 409 for mfa_already_enrolled. Following it makes
    // every account operation look as if it can answer 409, which is how a
    // checker starts reporting drift that is not there.
    const found = responders(`
func WriteAccountError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, ErrMfaAlreadyEnrolled):
		return WriteProblem(c, http.StatusConflict, "mfa_already_enrolled", "x")
	default:
		return err
	}
}

func handleListConsents(d Deps, c *echo.Context) error {
	consents, err := d.List(c.Request().Context())
	if err != nil {
		return WriteAccountError(c, err)
	}
	return c.JSON(http.StatusOK, consents)
}`)
    expect([...(found.get('handleListConsents')?.statuses ?? [])]).toEqual([200])
    expect(found.get('handleListConsents')?.unread).toEqual(['WriteAccountError'])
  })

  it('treats a helper that branches without errors.Is as a mapper when it takes an error', () => {
    // provisioning's writeError switches on isNotFound(err) / isConflict(err).
    // The dispatch is on the error value all the same.
    const found = responders(`
func (d Deps) writeError(c *echo.Context, err error) error {
	switch {
	case isNotFound(err):
		return support.WriteProblem(c, http.StatusNotFound, "provisioning_not_found", err.Error())
	default:
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
}

func (d Deps) handleGetConnection(c *echo.Context) error {
	conn, err := d.Repo.Find(c.Request().Context())
	if err != nil {
		return d.writeError(c, err)
	}
	return c.JSON(http.StatusOK, conn)
}`)
    expect([...(found.get('handleGetConnection')?.statuses ?? [])]).toEqual([200])
    expect(found.get('handleGetConnection')?.unread).toEqual(['writeError'])
  })

  it('records a write whose status is not a constant as unread', () => {
    const found = responders(`
func (h *Handler) handleScim(c *echo.Context) error {
	status := statusFor(err)
	return c.JSON(status, body)
}`)
    expect([...(found.get('handleScim')?.statuses ?? [])]).toEqual([])
    expect(found.get('handleScim')?.unread).toEqual(['JSON'])
  })

  it('records a handler that hands the raw response away as unread', () => {
    // /metrics lets the Prometheus handler write the whole response. Its 200
    // never appears as a status argument anywhere this reader can see.
    const found = responders(`
func (d Deps) handleMetrics(c *echo.Context) error {
	if d.MetricsHandler == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}
	d.MetricsHandler.ServeHTTP(c.Response(), c.Request())
	return nil
}`)
    expect([...(found.get('handleMetrics')?.statuses ?? [])]).toEqual([503])
    expect(found.get('handleMetrics')?.unread).toEqual(['ServeHTTP'])
  })

  it('does not mistake reading a header for handing the response away', () => {
    const found = responders(`
func (d Deps) handleGet(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(http.StatusOK, res)
}`)
    expect(found.get('handleGet')?.unread).toEqual([])
  })

  it('takes the constant at the call site even when the callee dispatches', () => {
    // writeScimError(c, http.StatusForbidden, ...) is settled by its caller; the
    // callee's own `c.JSON(status, ...)` says nothing more.
    const found = responders(`
func (h *Handler) writeScimError(c *echo.Context, status int, detail, scimType string) error {
	return c.JSON(status, domain.NewScimError(strconv.Itoa(status), detail, scimType))
}

func (h *Handler) handleDelete(c *echo.Context) error {
	if !allowed {
		return h.writeScimError(c, http.StatusForbidden, "denied", "")
	}
	return c.NoContent(http.StatusNoContent)
}`)
    expect([...(found.get('handleDelete')?.statuses ?? [])].sort()).toEqual([204, 403])
    expect(found.get('handleDelete')?.unread).toEqual([])
  })
})

const document = (
  operationId: string,
  statuses: number[],
  path = '/api/admin/v1/things',
  method = 'get',
): OpenAPIDocument => ({
  paths: {
    [path]: {
      [method]: {
        operationId,
        responses: Object.fromEntries(statuses.map((status) => [String(status), {}])),
      },
    },
  },
})

const run = (
  doc: OpenAPIDocument,
  source: string,
  routes = 'g.GET("/api/admin/v1/things", d.handleThings)',
): StatusDriftResult =>
  diffStatusCodes(doc, [
    { path: 'routes.go', source: routes },
    { path: 'h.go', source },
  ])

describe('S1: the contract does not declare a status the operation writes', () => {
  it('reports a status the handler writes but the contract omits', () => {
    const result = run(
      document('ListThings', [200]),
      `func (d Deps) handleThings(c *echo.Context) error {
	if missing {
		return support.WriteProblem(c, http.StatusNotFound, "thing_not_found", "x")
	}
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings.map((finding) => finding.key)).toEqual(['S1 ListThings'])
    expect(result.findings[0]?.message).toContain('404')
  })

  it('reports the 401 a guard writes beside the 403 the contract already declares', () => {
    // This is the shape the audit found on 217 operations: one guard, two
    // branches, one of them declared.
    const result = run(
      document('ListThings', [200, 403]),
      `func (d Deps) WriteAdminAccessError(c *echo.Context, err error) error {
	if errors.Is(err, ErrAdminAuthenticationRequired) {
		return WriteProblem(c, http.StatusUnauthorized, "authentication_required", "x")
	}
	return WriteProblem(c, http.StatusForbidden, "access_denied", "y")
}

func (d Deps) handleThings(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings).toHaveLength(1)
    expect(result.findings[0]?.message).toContain('401')
  })

  it('says nothing when the declaration matches', () => {
    const result = run(
      document('ListThings', [200, 404]),
      `func (d Deps) handleThings(c *echo.Context) error {
	if missing {
		return support.WriteProblem(c, http.StatusNotFound, "thing_not_found", "x")
	}
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings).toEqual([])
  })

  it('does not ask for the 500 the shared error handler writes', () => {
    // Every operation can reach it and no caller branches on it differently, so
    // docs/api-rules.md keeps it out of the per-operation declaration.
    const result = run(
      document('ListThings', [200]),
      `func (d Deps) handleThings(c *echo.Context) error {
	if err != nil {
		return support.WriteServerError(c, err)
	}
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings).toEqual([])
  })

  it('asks for a 5xx the handler writes with an error code of its own', () => {
    // 503 webauthn_unavailable is this operation's outcome, not the pipeline's.
    const result = run(
      document('ListThings', [200]),
      `func (d Deps) handleThings(c *echo.Context) error {
	if !configured {
		return support.WriteProblem(c, http.StatusServiceUnavailable, "webauthn_unavailable", "x")
	}
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings[0]?.message).toContain('503')
  })
})

describe('S2: the contract declares a status nothing writes', () => {
  it('reports an over-declared status when every writer was read', () => {
    const result = run(
      document('ListThings', [200, 400]),
      `func (d Deps) handleThings(c *echo.Context) error {
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings.map((finding) => finding.key)).toEqual(['S2 ListThings'])
    expect(result.findings[0]?.message).toContain('400')
  })

  it('stays silent when a writer was left unread', () => {
    // The 409 may well be reachable through the mapper. Reporting it as
    // over-declared would be answering a question the reader never asked.
    const result = run(
      document('ListThings', [200, 409]),
      `func WriteAccountError(c *echo.Context, err error) error {
	if errors.Is(err, ErrMfaAlreadyEnrolled) {
		return WriteProblem(c, http.StatusConflict, "mfa_already_enrolled", "x")
	}
	return err
}

func (d Deps) handleThings(c *echo.Context) error {
	things, err := d.List(c.Request().Context())
	if err != nil {
		return WriteAccountError(c, err)
	}
	return c.JSON(http.StatusOK, things)
}`,
    )
    expect(result.findings).toEqual([])
    expect(result.unread.map((entry) => entry.operationId)).toEqual(['ListThings'])
  })

  it('never reports a status the shared error handler can produce', () => {
    // 401, 403 and 422 reach the client through ErrorHandler on a bare `return
    // err`, which no reading of the handler's text can rule out.
    const result = run(
      document('ListThings', [200, 401, 403, 422, 400]),
      `func (d Deps) handleThings(c *echo.Context) error {
	return c.JSON(http.StatusOK, res)
}`,
    )
    expect(result.findings).toHaveLength(1)
    expect(result.findings[0]?.message).toContain('400')
    expect(result.findings[0]?.message).not.toContain('422')
  })
})

describe('coverage', () => {
  it('reports an operation whose route names no handler as unresolved', () => {
    const result = run(document('ListThings', [200]), 'package x', 'package x')
    expect(result.unresolved).toEqual([
      { operationId: 'ListThings', reason: 'route-not-found', detail: 'GET /api/admin/v1/things' },
    ])
    expect(result.findings).toEqual([])
  })

  it('reads nothing for a handler name two packages both define', () => {
    // Merging the two bodies would attribute one package's 404 to the other
    // package's operation, which reads exactly like real drift.
    const result = run(
      document('ListThings', [200]),
      `func (d Deps) handleThings(c *echo.Context) error {
	return c.JSON(http.StatusOK, res)
}

func (o Other) handleThings(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "thing_not_found", "x")
}`,
    )
    expect(result.findings).toEqual([])
    expect(result.unresolved).toEqual([
      { operationId: 'ListThings', reason: 'handler-ambiguous', detail: 'handleThings' },
    ])
  })

  it('separates a route it could not find from a writer it could not read', () => {
    const result = run(
      document('ListThings', [200, 409]),
      `func (d Deps) handleThings(c *echo.Context) error {
	things, err := d.List(c.Request().Context())
	if err != nil {
		return d.writeError(c, err)
	}
	return c.JSON(http.StatusOK, things)
}

func (d Deps) writeError(c *echo.Context, err error) error {
	return support.WriteProblem(c, http.StatusConflict, "conflict", "x")
}`,
    )
    expect(result.unresolved).toEqual([])
    expect(result.unread).toEqual([{ operationId: 'ListThings', writers: ['writeError'] }])
  })
})
