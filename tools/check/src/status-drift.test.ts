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

const problemDocument = (operationId: string, codes: string[]): OpenAPIDocument => ({
  paths: {
    '/api/admin/v1/things': {
      post: {
        operationId,
        responses: {
          '200': {},
          '403': {
            content: {
              'application/problem+json': {
                schema: { $ref: '#/components/schemas/ThingsError403Body' },
              },
            },
          },
        },
      },
    },
  },
  components: {
    schemas: {
      ThingsError403Body: {
        anyOf: codes.map((code) => ({ $ref: `#/components/schemas/${code}` })),
      },
      ...Object.fromEntries(
        codes.map((code) => [
          code,
          { description: `RFC 9457 Problem Details with type = urn:idmagic:error:${code}.` },
        ]),
      ),
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
    // docs/design/application/api-guidelines.md keeps it out of the per-operation declaration.
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

describe('B1: the declared 403 body covers guard refusal codes', () => {
  const apiTokenGuard = `func RequireAdmin(c *echo.Context) error {
	if tokenScopeMissing {
		return WriteProblem(c, http.StatusForbidden, "insufficient_scope", "The required scope is missing.")
	}
	return WriteProblem(c, http.StatusForbidden, "access_denied", "Administrator privileges are required.")
}

func (d Deps) handleThings(c *echo.Context) error {
	if err := RequireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`

  const browserGuard = `func VerifyBrowserRequest(c *echo.Context) error {
	if badOrigin {
		return WriteProblem(c, http.StatusForbidden, "invalid_origin", "The request origin does not match.")
	}
	return WriteProblem(c, http.StatusForbidden, "csrf_failed", "CSRF validation failed.")
}

func (d Deps) handleThings(c *echo.Context) error {
	if err := VerifyBrowserRequest(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`

  it('reports a guard error code omitted from the declared 403 body', () => {
    // REQ-APITOKENS-004: API トークンのスコープ不足は、403 本文に
    // insufficient_scope として宣言されなければならない。
    const result = run(
      problemDocument('UpdateThings', ['access_denied']),
      apiTokenGuard,
      'g.POST("/api/admin/v1/things", d.handleThings)',
    )
    expect(result.findings.map((finding) => finding.key)).toEqual(['B1 UpdateThings'])
    expect(result.findings[0]?.message).toContain('insufficient_scope')
  })

  it('follows browser guard error codes into the declared 403 body', () => {
    // REQ-PLATFORM-004: browser guard の origin と CSRF の拒否コードを、
    // 呼び出し元 operation の宣言まで伝播する。
    const result = run(
      problemDocument('UpdateThings', ['invalid_origin', 'csrf_failed']),
      browserGuard,
      'g.POST("/api/admin/v1/things", d.handleThings)',
    )
    expect(result.findings).toEqual([])
  })

  it('matches the repository 403 bodies to guard error codes', () => {
    // REQ-APITOKENS-004: operation scope を持たないトークンは
    // insufficient_scope、管理者でない対話セッションは access_denied になる。
    const result = run(
      problemDocument('UpdateThings', ['access_denied', 'insufficient_scope']),
      apiTokenGuard,
      'g.POST("/api/admin/v1/things", d.handleThings)',
    )
    expect(result.findings).toEqual([])
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

  it('fails an operation whose handler name two definitions share', () => {
    // 経路の登録はハンドラーの定義位置を持たないため、どちらを読むかを決められない。
    // 件数に数えるだけでは検査が通るので、失敗として報告する。
    const result = run(
      document('ListThings', [200]),
      `func (d Deps) handleThings(c *echo.Context) error {
	return c.JSON(http.StatusOK, res)
}

func (o Other) handleThings(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "thing_not_found", "x")
}`,
    )
    expect(result.findings.map((finding) => finding.key)).toEqual(['A1 ListThings'])
    expect(result.findings[0]?.message).toContain('handleThings')
    expect(result.unresolved).toEqual([])
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

describe('same-named guards in different packages', () => {
  const tokensGuard = `func (d Deps) requireAdmin(c *echo.Context) error {
	return WriteProblem(c, http.StatusUnauthorized, "authentication_required", "x")
}`
  const federationGuard = `func requireAdmin(d Deps, c *echo.Context) error {
	return WriteProblem(c, http.StatusForbidden, "access_denied", "y")
}`
  const twoPackages = (tokensHandler: string, federationHandler: string) => [
    {
      path: 'backend/apitokens/routes.go',
      source: 'g.GET("/api/admin/v1/tokens", d.handleTokens)',
    },
    { path: 'backend/apitokens/guard.go', source: tokensGuard },
    { path: 'backend/apitokens/handler.go', source: tokensHandler },
    {
      path: 'backend/federation/routes.go',
      source: 'g.GET("/api/admin/v1/federation", d.handleFederation)',
    },
    { path: 'backend/federation/guard.go', source: federationGuard },
    { path: 'backend/federation/handler.go', source: federationHandler },
  ]
  const contract: OpenAPIDocument = {
    paths: {
      '/api/admin/v1/tokens': { get: { operationId: 'ListTokens', responses: { '200': {} } } },
      '/api/admin/v1/federation': {
        get: { operationId: 'GetFederation', responses: { '200': {} } },
      },
    },
  }

  it('attributes each guard to the handler in its own package', () => {
    // wi-551 では ApiTokens と federation の requireAdmin が同名だったため、
    // どちらの 401 と 403 も読まれず、宣言漏れが検査を通っていた。
    const found = collectResponders(
      twoPackages(
        `func (d Deps) handleTokens(c *echo.Context) error {
	if err := d.requireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
        `func (d Deps) handleFederation(c *echo.Context) error {
	if err := requireAdmin(d, c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
      ),
    )
    expect([...(found.get('handleTokens')?.statuses ?? [])].sort()).toEqual([200, 401])
    expect([...(found.get('handleFederation')?.statuses ?? [])].sort()).toEqual([200, 403])
    expect(found.get('handleTokens')?.unread).toEqual([])
    expect(found.get('handleFederation')?.unread).toEqual([])
  })

  it('reports the undeclared statuses each package guard writes', () => {
    const result = diffStatusCodes(
      contract,
      twoPackages(
        `func (d Deps) handleTokens(c *echo.Context) error {
	if err := d.requireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
        `func (d Deps) handleFederation(c *echo.Context) error {
	if err := requireAdmin(d, c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
      ),
    )
    expect(result.findings.map((finding) => finding.message)).toEqual([
      'S1 ListTokens: handleTokens writes 401, which the contract does not declare',
      'S1 GetFederation: handleFederation writes 403, which the contract does not declare',
    ])
  })

  it('fails a guard call through a field whose definition it cannot settle', () => {
    // d.Auth の型は字面から決まらないため、別パッケージの 2 つの定義のどちらも候補に残る。
    // 読み飛ばすと 401 と 403 の帰属が黙って欠けるので、候補の位置を添えて失敗させる。
    const result = diffStatusCodes(
      contract,
      twoPackages(
        `func (d Deps) handleTokens(c *echo.Context) error {
	if err := d.requireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
        `func (d Deps) handleFederation(c *echo.Context) error {
	if err := d.Auth.requireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
      ).map((file) =>
        file.path === 'backend/federation/guard.go'
          ? { ...file, path: 'backend/shared/guard.go' }
          : file,
      ),
    )
    expect(result.findings.map((finding) => finding.key)).toEqual([
      'S1 ListTokens',
      'A1 GetFederation',
    ])
    expect(result.findings[1]?.message).toBe(
      'A1 GetFederation: handleFederation calls requireAdmin, which is defined at ' +
        'backend/apitokens/guard.go, backend/shared/guard.go; the reader cannot tell which one runs',
    )
  })

  it('reads the one definition a qualified call can reach', () => {
    // 候補が 1 つしかなければ、受け手の型を決められなくても読み違えない。
    const found = collectResponders([
      { path: 'backend/shared/guard.go', source: federationGuard },
      {
        path: 'backend/federation/handler.go',
        source: `func (d Deps) handleFederation(c *echo.Context) error {
	if err := d.Auth.requireAdmin(d, c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
      },
    ])
    expect([...(found.get('handleFederation')?.statuses ?? [])].sort()).toEqual([200, 403])
    expect(found.get('handleFederation')?.ambiguous).toEqual([])
  })

  it('keeps an ambiguous error mapper as unread rather than failing', () => {
    // どちらの定義もエラー値で分岐するなら、解決できても読まない。結果は同じなので失敗にしない。
    const mapper = (code: string) => `func writeError(c *echo.Context, err error) error {
	return WriteProblem(c, http.StatusConflict, "${code}", "x")
}`
    const found = collectResponders([
      { path: 'backend/a/errors.go', source: mapper('a_conflict') },
      { path: 'backend/b/errors.go', source: mapper('b_conflict') },
      {
        path: 'backend/c/handler.go',
        source: `func (d Deps) handleThings(c *echo.Context) error {
	return d.Errors.writeError(c, err)
}`,
      },
    ])
    expect(found.get('handleThings')?.unread).toEqual(['writeError'])
    expect(found.get('handleThings')?.ambiguous).toEqual([])
  })
})

describe('an ambiguous call behind a guard', () => {
  it('fails the operation and reports no over-declaration', () => {
    // 曖昧な呼び出しは前提判定の連鎖を通じてハンドラーへ届く。どちらの定義が
    // 400 を書くかは分からないので、400 を宣言過剰とは呼ばない。
    const check = (code: string) => `func (s Session) checkAdmin(c *echo.Context) error {
	return WriteProblem(c, http.StatusForbidden, "${code}", "x")
}`
    const result = diffStatusCodes(document('ListThings', [200, 400]), [
      { path: 'backend/things/routes.go', source: 'g.GET("/api/admin/v1/things", d.handleThings)' },
      { path: 'backend/session/a/check.go', source: check('access_denied') },
      { path: 'backend/session/b/check.go', source: check('insufficient_scope') },
      {
        path: 'backend/things/handler.go',
        source: `func (d Deps) requireAdmin(c *echo.Context) error {
	return d.Session.checkAdmin(c)
}

func (d Deps) handleThings(c *echo.Context) error {
	if err := d.requireAdmin(c); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}`,
      },
    ])
    expect(result.findings.map((finding) => finding.key)).toEqual(['A1 ListThings'])
    expect(result.findings[0]?.message).toContain(
      'checkAdmin, which is defined at backend/session/a/check.go, backend/session/b/check.go',
    )
  })
})
