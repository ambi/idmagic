import { describe, expect, it } from 'bun:test'
import {
  checkContractRefusalsAreDeclared,
  checkSecurityGuards,
  browserRefusalTypesNamedByPlatformScenario,
  insufficientScopeTypeNamedByApiTokenScenario,
  contractRefusalsOfStateChanges,
  errorTypesNamedByScenarios,
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

describe('errorTypesNamedByScenarios', () => {
  const scenarios = (outcomes: string[]) =>
    [
      '# Feature: Jobs',
      '',
      '## Rule: REQ-JOBS-012 an administrator lists their tenant',
      '',
      '### Example: EX-JOBS-012-01 refusal',
      '',
      '- When the administrator lists jobs',
      ...outcomes.map((outcome, index) => '- ' + (index === 0 ? 'Then' : 'And') + ' ' + outcome),
      '',
    ].join('\n')

  it('collects an error type from an outcome step', () => {
    expect([...errorTypesNamedByScenarios(scenarios(['AccessDeniedError is returned']))]).toEqual([
      'AccessDeniedError',
    ])
  })

  // The type is what R4 joins on, and it is written on both sides: in TypeSpec
  // as the 403 body, in the scenario as the name of what answers. Whether the
  // step also reads as a refusal in prose is not part of that join.
  it('collects a type from a step that carries no refusal vocabulary', () => {
    expect([
      ...errorTypesNamedByScenarios(scenarios(['JobClaimExpiredError is returned to the caller'])),
    ]).toEqual(['JobClaimExpiredError'])
  })

  it('reads every outcome in an example', () => {
    expect(
      [
        ...errorTypesNamedByScenarios(scenarios(['AccessDeniedError', 'InvalidRequestError'])),
      ].sort(),
    ).toEqual(['AccessDeniedError', 'InvalidRequestError'])
  })

  it('ignores a step that names no error type', () => {
    expect([
      ...errorTypesNamedByScenarios(scenarios(['the worker claims it and it succeeds'])),
    ]).toEqual([])
  })
})

describe('browserRefusalTypesNamedByPlatformScenario', () => {
  const platformScenarios = (ruleId: string) =>
    [
      '# Feature: Platform',
      '',
      `## Rule: ${ruleId} browser requests fail closed`,
      '',
      '### Example: EX-PLATFORM-004-01 invalid origin',
      '',
      '- When a browser changes state from another origin',
      '- Then InvalidOriginError is returned and state is unchanged',
      '',
      '### Example: EX-PLATFORM-004-02 invalid CSRF proof',
      '',
      '- When a browser changes state without a matching CSRF proof',
      '- Then CsrfFailedError is returned and state is unchanged',
      '',
    ].join('\n')

  it('names the shared refusals only from REQ-PLATFORM-004', () => {
    expect([
      ...browserRefusalTypesNamedByPlatformScenario(platformScenarios('REQ-PLATFORM-004')),
    ]).toEqual(['InvalidOriginError', 'CsrfFailedError'])
  })

  it('does not accept the same prose under another requirement ID', () => {
    expect([
      ...browserRefusalTypesNamedByPlatformScenario(platformScenarios('REQ-PLATFORM-099')),
    ]).toEqual([])
  })
})

describe('insufficientScopeTypeNamedByApiTokenScenario', () => {
  const apiTokenScenarios = (ruleId: string) =>
    [
      '# Feature: API tokens',
      '',
      `## Rule: ${ruleId} scopes fail closed`,
      '',
      '### Example: EX-APITOKENS-004-02 missing scope',
      '',
      '- When a client calls a management API without its required scope',
      '- Then InsufficientScopeError is returned and management state is unchanged',
      '',
    ].join('\n')

  it('names the shared refusal only from REQ-APITOKENS-004', () => {
    expect([
      ...insufficientScopeTypeNamedByApiTokenScenario(apiTokenScenarios('REQ-APITOKENS-004')),
    ]).toEqual(['InsufficientScopeError'])
  })

  it('does not accept the same prose under another requirement ID', () => {
    expect([
      ...insufficientScopeTypeNamedByApiTokenScenario(apiTokenScenarios('REQ-APITOKENS-099')),
    ]).toEqual([])
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
  const scenarioDocument = (given: string, outcome: string) =>
    [
      '# Feature: Signing keys',
      '',
      '## Rule: REQ-SIGNINGKEYS-001 rotation keeps the previous kid on the JWKS',
      '',
      '### Example: EX-SIGNINGKEYS-001-01 rotation',
      '',
      `- Given ${given}`,
      '- When the administrator rotates the signing key',
      `- Then ${outcome}`,
      '',
    ].join('\n')

  it('rejects a promised refusal no scenario declares', () => {
    const scenarios = scenarioDocument('a current key exists', 'both kids are on the JWKS')
    const findings = checkContractRefusalsAreDeclared(
      'signing-keys',
      contract,
      errorTypesNamedByScenarios(scenarios),
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.rule).toBe('R4')
    expect(findings[0]?.message).toContain('RotateTenantSigningKey')
    expect(findings[0]?.path).toBe('docs/contexts/signing-keys/scenarios.feature.md')
  })

  it('accepts the refusal once a scenario declares it', () => {
    const scenarios = scenarioDocument(
      'an operator is not an administrator',
      'AccessDeniedError で拒否され、有効な鍵は変わらない',
    )
    expect(
      checkContractRefusalsAreDeclared(
        'signing-keys',
        contract,
        errorTypesNamedByScenarios(scenarios),
      ),
    ).toEqual([])
  })

  // The error type has to be named where the refusal is, not anywhere in the
  // document: a success step mentioning the type says nothing about when it fires.
  it('does not accept the type named outside a refusal', () => {
    const scenarios = scenarioDocument(
      'the AccessDeniedError body is documented',
      'both kids are on the JWKS',
    )
    expect(
      checkContractRefusalsAreDeclared(
        'signing-keys',
        contract,
        errorTypesNamedByScenarios(scenarios),
      ),
    ).toHaveLength(1)
  })
})
