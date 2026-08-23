import { describe, expect, it } from 'bun:test'
import {
  checkContractRefusalsAreDeclared,
  checkRefusalCoverage,
  checkSecurityGuards,
  contractRefusalsOfStateChanges,
  declaredRefusalTypes,
  refusalScenarioIds,
} from './security-controls.ts'

// The writers every fixture needs, because the rule seeds on echo's own
// response methods and derives the rest. `WriteProblem` is not privileged: it
// reaches the set through `NoStoreJSON` down to `c.JSON`, the same way any
// helper someone writes tomorrow will.
const supportWriters = {
  path: 'backend/shared/http/support_http/problem.go',
  source: [
    'func WriteProblem(c *echo.Context, status int, code, detail string) error {',
    '\treturn NoStoreJSON(c, status, Problem{Type: code})',
    '}',
    '',
    'func NoStoreJSON(c *echo.Context, status int, body any) error {',
    '\treturn c.JSON(status, body)',
    '}',
  ].join('\n'),
}

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
    const findings = checkSecurityGuards([supportWriters, fallThroughGuard, guardCaller])
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
    expect(checkSecurityGuards([supportWriters, fixed, guardCaller])).toEqual([])
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
    expect(checkSecurityGuards([supportWriters, handler])).toEqual([])
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
    expect(checkSecurityGuards([supportWriters, test, writer])).toEqual([])
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
    const findings = checkSecurityGuards([supportWriters, guard, guardCaller, discarding])
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

  // What actually shipped, and what naming `WriteProblem` directly could not
  // see: the guard returns a helper, and the helper is the one handing back
  // what writing the response returned. Two non-administrators reached a model
  // version and a workflow listing through this, each reading its own 403.
  const adminAccessWriter = {
    path: 'backend/shared/http/support_http/auth.go',
    source: [
      'func (a *Authenticator) WriteAdminAccessError(c *echo.Context, err error) error {',
      '\tif errors.Is(err, ErrAdminAccessDenied) {',
      '\t\treturn WriteProblem(c, http.StatusForbidden, "access_denied", "no")',
      '\t}',
      '\treturn err',
      '}',
    ].join('\n'),
  }

  const indirectGuard = {
    path: 'backend/authorization/handlers_http/routes.go',
    source: [
      'func (d Deps) requireAuthorizationAdmin(c *echo.Context) error {',
      '\tif _, err := d.RequireAdmin(c); err != nil {',
      '\t\treturn d.WriteAdminAccessError(c, err)',
      '\t}',
      '\treturn nil',
      '}',
      '',
      'func (d Deps) handlePutAuthorizationModel(c *echo.Context) error {',
      '\tif err := d.requireAuthorizationAdmin(c); err != nil {',
      '\t\treturn err',
      '\t}',
      '\treturn d.saveModel(c)',
      '}',
    ].join('\n'),
  }

  it('rejects a guard that returns a writer one call away', () => {
    const findings = checkSecurityGuards([supportWriters, adminAccessWriter, indirectGuard])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R1')
    expect(findings[0]?.message).toContain('requireAuthorizationAdmin')
    // The chain is named so the fix is obvious from the finding alone.
    expect(findings[0]?.message).toContain('WriteAdminAccessError')
    expect(findings[0]?.message).toContain('WriteProblem')
  })

  // One level was the defect that shipped; there is nothing about the shape
  // that stops at one. A per-context wrapper around the shared writer is the
  // obvious next step, and it must not buy an exemption.
  it('rejects a guard that returns a writer two calls away', () => {
    const contextWriter = {
      path: 'backend/idgovernance/handlers_http/errors.go',
      source: [
        'func (d Deps) writeWorkflowAccessError(c *echo.Context, err error) error {',
        '\treturn d.WriteAdminAccessError(c, err)',
        '}',
      ].join('\n'),
    }
    const guard = {
      path: 'backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go',
      source: [
        'func (d Deps) requireWorkflowAdmin(c *echo.Context) error {',
        '\tif _, err := d.RequireAdmin(c); err != nil {',
        '\t\treturn d.writeWorkflowAccessError(c, err)',
        '\t}',
        '\treturn nil',
        '}',
        '',
        'func (d Deps) handleListLifecycleWorkflows(c *echo.Context) error {',
        '\tif err := d.requireWorkflowAdmin(c); err != nil {',
        '\t\treturn err',
        '\t}',
        '\treturn support.NoStoreJSON(c, http.StatusOK, workflows)',
        '}',
      ].join('\n'),
    }
    const findings = checkSecurityGuards([supportWriters, adminAccessWriter, contextWriter, guard])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R1')
    expect(findings[0]?.message).toContain('requireWorkflowAdmin')
    expect(findings[0]?.message).toContain('writeWorkflowAccessError')
  })

  // The fix wi-391 applied: the writer returns a sentinel, so nothing that
  // wraps it hands a caller a nil. The guards above become correct without
  // being touched, which is the point of judging the writer and not the name.
  it('accepts the same guards once the writer returns a sentinel', () => {
    const refusing = {
      ...adminAccessWriter,
      source: [
        'func (a *Authenticator) WriteAdminAccessError(c *echo.Context, err error) error {',
        '\tif errors.Is(err, ErrAdminAccessDenied) {',
        '\t\treturn refused(WriteProblem(c, http.StatusForbidden, "access_denied", "no"))',
        '\t}',
        '\treturn err',
        '}',
        '',
        'func refused(writeErr error) error {',
        '\tif writeErr != nil {',
        '\t\treturn writeErr',
        '\t}',
        '\treturn ErrAdminAccessRefused',
        '}',
      ].join('\n'),
    }
    expect(checkSecurityGuards([supportWriters, refusing, indirectGuard])).toEqual([])
  })

  // The seed is echo's own writers, so a guard that skips the repository's
  // helpers and writes the refusal itself is judged the same way.
  it('rejects a guard that returns an echo write directly', () => {
    const guard = {
      path: 'backend/shared/http/support_http/csrf.go',
      source: [
        'func (d Deps) VerifyBrowserRequest(c *echo.Context) error {',
        '\tif origin != issuer {',
        '\t\treturn c.NoContent(http.StatusForbidden)',
        '\t}',
        '\treturn nil',
        '}',
      ].join('\n'),
    }
    const findings = checkSecurityGuards([guard, guardCaller])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R1')
  })

  // `JSON` on something that is not the request context is an encoder, a
  // fixture, a DTO -- not a response going out.
  it('does not treat a writer-shaped method on another receiver as a write', () => {
    const guard = {
      path: 'backend/shared/http/support_http/csrf.go',
      source: [
        'func (d Deps) VerifyBrowserRequest(c *echo.Context) error {',
        '\tif origin != issuer {',
        '\t\treturn d.encoder.JSON(origin)',
        '\t}',
        '\treturn nil',
        '}',
      ].join('\n'),
    }
    expect(checkSecurityGuards([guard, guardCaller])).toEqual([])
  })
})

describe('refusalScenarioIds', () => {
  it('collects a refusal declared as an alternative', () => {
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

  // The refusal is the whole point of the scenario, so there is no alternative
  // to hang it on. Reading only ALT missed REQ-SIGNINGKEYS-009 entirely.
  it('collects a refusal declared as the outcome', () => {
    const scenarios = [
      '### REQ-SIGNINGKEYS-009: a tenant administrator cannot reach signing key health',
      '- ACTOR TenantAdministrator',
      '- WHEN "operator" calls the signing key health listing',
      '- THEN AccessDeniedError で拒否される',
    ].join('\n')
    expect(refusalScenarioIds(scenarios)).toEqual(['REQ-SIGNINGKEYS-009'])
  })

  // A tenant boundary is stated as what the caller sees, not as an error.
  it('collects a refusal phrased as an outcome rather than an error', () => {
    const scenarios = [
      '### REQ-IDGOVERNANCE-012: another tenant cannot reach the workflow',
      "- WHEN the administrator reads another tenant's workflow",
      '- THEN ワークフローは存在しないものとして扱われる',
    ].join('\n')
    expect(refusalScenarioIds(scenarios)).toEqual(['REQ-IDGOVERNANCE-012'])
  })

  it('leaves a scenario that declares no refusal alone', () => {
    const scenarios = [
      '### REQ-JOBS-002: a submitted job succeeds',
      '- WHEN a job is enqueued',
      '- THEN the worker claims it and it succeeds',
    ].join('\n')
    expect(refusalScenarioIds(scenarios)).toEqual([])
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

// The shape TypeSpec is generated in: the 403 body is a union named after the
// operation, and the decorators sit above the method.
const rotateKey = [
  'union RotateTenantSigningKeyError403Body {',
  '  IdMagic.Contract.AccessDeniedError,',
  '}',
  '',
  '@TypeSpec.OpenAPI.operationId("RotateTenantSigningKey")',
  '@route("/api/admin/v1/keys/rotate")',
  '@post',
  'op RotateTenantSigningKey(',
  '',
  '): RotateTenantSigningKeySuccess_200 | RotateTenantSigningKeyError403;',
].join('\n')

describe('contractRefusalsOfStateChanges', () => {
  it('reads the 403 body of a state-changing operation', () => {
    expect([...contractRefusalsOfStateChanges(rotateKey)]).toEqual([
      ['AccessDeniedError', ['RotateTenantSigningKey']],
    ])
  })

  // A refused read runs no risk of leaving an effect behind, and demanding a
  // scenario for every listing would drown the rule in paperwork.
  it('ignores a read', () => {
    const listKeys = rotateKey
      .replaceAll('RotateTenantSigningKey', 'ListAdminKeys')
      .replace('@post', '@get')
    expect([...contractRefusalsOfStateChanges(listKeys)]).toEqual([])
  })

  it('ignores a status other than 403', () => {
    const badRequest = rotateKey.replaceAll('403', '400')
    expect([...contractRefusalsOfStateChanges(badRequest)]).toEqual([])
  })
})

describe('checkContractRefusalsAreDeclared', () => {
  const contract = contractRefusalsOfStateChanges(rotateKey)

  it('rejects a promised refusal no scenario declares', () => {
    const scenarios = [
      '### REQ-SIGNINGKEYS-001: rotation keeps the previous kid on the JWKS',
      '- WHEN the administrator rotates the signing key',
      '- THEN both kids are on the JWKS',
    ].join('\n')
    const findings = checkContractRefusalsAreDeclared(
      'signing-keys',
      contract,
      declaredRefusalTypes(scenarios),
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R4')
    expect(findings[0]?.message).toContain('RotateTenantSigningKey')
    expect(findings[0]?.path).toBe('docs/contexts/signing-keys/scenarios.md')
  })

  it('accepts the refusal once a scenario declares it', () => {
    const scenarios = [
      '### REQ-SIGNINGKEYS-011: only an administrator rotates a signing key',
      '- WHEN "operator" rotates the signing key',
      '- THEN AccessDeniedError で拒否され、有効な鍵は変わらない',
    ].join('\n')
    expect(
      checkContractRefusalsAreDeclared('signing-keys', contract, declaredRefusalTypes(scenarios)),
    ).toEqual([])
  })

  // The error type has to be named where the refusal is, not anywhere in the
  // document: a success step mentioning the type says nothing about when it fires.
  it('does not accept the type named outside a refusal', () => {
    const scenarios = [
      '### REQ-SIGNINGKEYS-001: rotation keeps the previous kid on the JWKS',
      '- GIVEN the AccessDeniedError body is documented',
      '- THEN both kids are on the JWKS',
    ].join('\n')
    expect(
      checkContractRefusalsAreDeclared('signing-keys', contract, declaredRefusalTypes(scenarios)),
    ).toHaveLength(1)
  })
})
