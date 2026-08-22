import { describe, expect, it } from 'bun:test'
import {
  checkRefusalCoverage,
  checkSecurityGuards,
  refusalScenarioIds,
} from './security-controls.ts'

// The shape of the defect this check exists for: the guard writes its refusal
// and hands the caller a nil, so the caller runs on.
const fallThroughGuard = {
  path: 'backend/shared/http/support_http/csrf.go',
  source: [
    'func (d Deps) VerifyBrowserRequest(c *echo.Context) error {',
    '\tif origin != issuer {',
    '\t\treturn WriteProblem(c, http.StatusForbidden, "invalid_origin", "no")',
    '\t}',
    '\treturn nil',
    '}',
  ].join('\n'),
}

const guardCaller = {
  path: 'backend/jobs/handlers_http/admin_job_handler.go',
  source: [
    'func (d Deps) handleCancelJob(c *echo.Context) error {',
    '\tif err := d.VerifyBrowserRequest(c); err != nil {',
    '\t\treturn err',
    '\t}',
    '\treturn support.NoStoreJSON(c, http.StatusOK, nil)',
    '}',
  ].join('\n'),
}

describe('checkSecurityGuards', () => {
  it('rejects a guard that returns the result of writing its refusal', () => {
    const findings = checkSecurityGuards([fallThroughGuard, guardCaller])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R1')
    expect(findings[0]?.message).toContain('VerifyBrowserRequest')
  })

  it('accepts a guard that returns a sentinel after writing', () => {
    const fixed = {
      ...fallThroughGuard,
      source: [
        'func (d Deps) VerifyBrowserRequest(c *echo.Context) error {',
        '\tif origin != issuer {',
        '\t\t_ = WriteProblem(c, http.StatusForbidden, "invalid_origin", "no")',
        '\t\treturn ErrBrowserVerificationFailed',
        '\t}',
        '\treturn nil',
        '}',
      ].join('\n'),
    }
    expect(checkSecurityGuards([fixed, guardCaller])).toEqual([])
  })

  // A route handler's job is to write the response and return. It is only a
  // guard when a caller decides whether to continue from what it returns.
  it('leaves a handler that writes and returns alone', () => {
    const handler = {
      path: 'backend/audit/handlers_http/admin_audit_event_handler.go',
      source: [
        'func (d Deps) handleGetAdminAuditEvent(c *echo.Context) error {',
        '\tif missing {',
        '\t\treturn support.WriteProblem(c, http.StatusNotFound, "not_found", "no")',
        '\t}',
        '\treturn nil',
        '}',
      ].join('\n'),
    }
    expect(checkSecurityGuards([handler])).toEqual([])
  })

  // A test may call a writer inside `if err := ...` to assert what it returns.
  // That says nothing about how the product uses it.
  it('does not take a guard position from a test file', () => {
    const test = {
      path: 'backend/shared/http/support_http/auth_admin_test.go',
      source: [
        'func TestWriteAdminAccessError(t *testing.T) {',
        '\tif err := a.WriteAdminAccessError(c, other); err != nil {',
        '\t\tt.Fatal(err)',
        '\t}',
        '}',
      ].join('\n'),
    }
    const writer = {
      path: 'backend/shared/http/support_http/auth.go',
      source: [
        'func (a *Authenticator) WriteAdminAccessError(c *echo.Context, err error) error {',
        '\treturn WriteProblem(c, http.StatusForbidden, "access_denied", "no")',
        '}',
      ].join('\n'),
    }
    expect(checkSecurityGuards([test, writer])).toEqual([])
  })

  it('rejects discarding the result of something used as a guard elsewhere', () => {
    const discarding = {
      path: 'backend/other/handlers_http/routes.go',
      source: [
        'func (d Deps) handleSomething(c *echo.Context) error {',
        '\t_ = d.VerifyBrowserRequest(c)',
        '\treturn nil',
        '}',
      ].join('\n'),
    }
    // The guard has to be defined for R2 to recognise the name: matching on the
    // name alone would flag every discarded Save and Revoke in the repository.
    const guard = {
      path: 'backend/shared/http/support_http/csrf.go',
      source: [
        'func (d Deps) VerifyBrowserRequest(c *echo.Context) error {',
        '\t_ = WriteProblem(c, http.StatusForbidden, "invalid_origin", "no")',
        '\treturn ErrResponseWritten',
        '}',
      ].join('\n'),
    }
    const findings = checkSecurityGuards([guard, guardCaller, discarding])
    expect(findings.map((f) => f.rule)).toContain('R2')
  })

  it('does not flag a discarded call that merely shares a name with a guard', () => {
    const repository = {
      path: 'backend/saml/db_memory/saml_service_providers.go',
      source: ['func (r *Repo) prune() {', '\t_ = r.Save(ctx, sp)', '}'].join('\n'),
    }
    const guarded = {
      path: 'backend/saml/handlers_http/admin_service_provider_handler.go',
      source: [
        'func (d Deps) handleUpsert(c *echo.Context) error {',
        '\tif err := d.Save(c.Request().Context(), sp); err != nil {',
        '\t\treturn err',
        '\t}',
        '\treturn nil',
        '}',
      ].join('\n'),
    }
    expect(checkSecurityGuards([repository, guarded])).toEqual([])
  })
})

describe('refusalScenarioIds', () => {
  it('collects the scenarios whose alternatives declare a refusal', () => {
    const scenarios = [
      '### REQ-JOBS-012: an administrator lists their tenant',
      '- WHEN the administrator lists jobs',
      '  - ALT the caller holds no admin role → AccessDeniedError で拒否される',
      '- THEN the page is returned',
      '',
      '### REQ-JOBS-020: the page is ordered newest first',
      '- WHEN the administrator lists jobs',
      '  - ALT two jobs share an instant → the id breaks the tie',
      '- THEN the page is ordered',
    ].join('\n')
    expect(refusalScenarioIds(scenarios)).toEqual(['REQ-JOBS-012'])
  })
})

describe('checkRefusalCoverage', () => {
  it('accepts a declared refusal that a test names', () => {
    expect(checkRefusalCoverage(['REQ-JOBS-012'], new Set(['REQ-JOBS-012']), [])).toEqual([])
  })

  it('rejects a declared refusal that no test names', () => {
    const findings = checkRefusalCoverage(['REQ-JOBS-012'], new Set(), [])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R3')
    expect(findings[0]?.message).toContain('REQ-JOBS-012')
  })

  it('tolerates a refusal carried on the known-debt list', () => {
    expect(checkRefusalCoverage(['REQ-JOBS-012'], new Set(), ['REQ-JOBS-012'])).toEqual([])
  })

  // The list is a ratchet: once a refusal has a test, keeping it listed would
  // let the debt silently come back.
  it('requires an entry to be removed once it has a test', () => {
    const findings = checkRefusalCoverage(['REQ-JOBS-012'], new Set(['REQ-JOBS-012']), [
      'REQ-JOBS-012',
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('only shrinks')
  })

  it('requires an entry to be removed once it declares no refusal', () => {
    const findings = checkRefusalCoverage([], new Set(), ['REQ-JOBS-012'])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('no longer declares one')
  })
})
