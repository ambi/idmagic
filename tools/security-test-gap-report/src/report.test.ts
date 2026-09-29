import { describe, expect, test } from 'bun:test'

import { effectEvidenceIn, reportSecurityTestGaps } from './report.ts'

describe('effectEvidenceIn', () => {
  test('does not count a refusal response by itself', () => {
    const body = `
      refused := send(t, http.MethodPost, "/items", request)
      if refused.Code != http.StatusForbidden {
        t.Fatalf("status=%d, want 403", refused.Code)
      }
      if problemCode(t, refused) != "access_denied" {
        t.Fatal("wrong problem code")
      }
    `

    expect(effectEvidenceIn(body)).toEqual([])
  })

  test('recognizes a state snapshot compared after the refusal', () => {
    const body = `
      before := fixture.user(t, alice)
      refused := send(t, http.MethodPatch, "/users/alice", request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      after := fixture.user(t, alice)
      if after.Name != before.Name { t.Fatal("user changed") }
    `

    expect(effectEvidenceIn(body)).toContain('state comparison')
  })

  test('recognizes state snapshot names qualified by the observed effect', () => {
    const body = `
      beforeInvalid := fixture.profile(t, alice)
      refused := send(t, http.MethodPatch, "/users/alice", request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      afterInvalid := fixture.profile(t, alice)
      if afterInvalid.Name != beforeInvalid.Name { t.Fatal("profile changed") }
    `

    expect(effectEvidenceIn(body)).toContain('state comparison')
  })

  test('recognizes a recorded external effect assertion', () => {
    const body = `
      refused := send(t, http.MethodPost, "/notifications", request)
      if refused.Code != http.StatusBadRequest { t.Fatal("want refusal") }
      if len(server.sender.Sent) != 0 { t.Fatal("message was sent") }
    `

    expect(effectEvidenceIn(body)).toContain('recorded effect')
  })

  test('recognizes empty state listings and local event records', () => {
    const body = `
      refused := send(t, http.MethodPost, "/streams", request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      if len(view.Streams) != 0 { t.Fatal("stream was created") }
      if len(events) != 0 { t.Fatal("event was emitted") }
    `

    expect(effectEvidenceIn(body)).toContain('recorded effect')
  })

  test('recognizes absence of a protected representation in a refused read', () => {
    const body = `
      refused := getAs(server, regularUser, "/applications")
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      if strings.Contains(refused.Body.String(), applicationID) {
        t.Fatal("refusal leaked the protected application")
      }
    `

    expect(effectEvidenceIn(body)).toContain('protected representation absent')
  })

  test('recognizes an inline response body checked for a protected representation', () => {
    const body = `
      refused := getAs(server, regularUser, "/applications")
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      if body := refused.Body.String(); strings.Contains(body, applicationID) {
        t.Fatal("refusal leaked the application")
      }
    `

    expect(effectEvidenceIn(body)).toContain('protected representation absent')
  })

  test('recognizes an asserted observation returned by a fixture helper', () => {
    const body = `
      refused := fixture.send(t, request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      agent := fixture.agent(t, agentID)
      if agent.Status != domain.AgentStatusActive { t.Fatal("agent changed") }
    `

    expect(effectEvidenceIn(body)).toContain('asserted observation')
  })

  test('recognizes an inline assertion over a fixture observation', () => {
    const body = `
      refused := fixture.send(t, request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      if count := adminGroupCount(t, server); count != 0 {
        t.Fatalf("created %d groups", count)
      }
    `

    expect(effectEvidenceIn(body)).toContain('asserted observation')
  })

  test('recognizes an asserted representation from a two-value read helper', () => {
    const body = `
      refused := fixture.send(t, request)
      if refused.Code != http.StatusBadRequest { t.Fatal("want refusal") }
      _, resource := doScimGet(t, server, resourcePath)
      if resource["userName"] != before["userName"] { t.Fatal("user changed") }
    `

    expect(effectEvidenceIn(body)).toContain('asserted observation')
  })

  test('recognizes an asserted field returned by an observation helper', () => {
    const body = `
      refused := fixture.send(t, request)
      if refused.Code != http.StatusBadRequest { t.Fatal("want refusal") }
      saved := readSamlDetail(t, server, applicationID).Saml
      if saved.WantAuthnRequestsSigned { t.Fatal("configuration changed") }
    `

    expect(effectEvidenceIn(body)).toContain('asserted observation')
  })

  test('recognizes an effect-specific assertion helper', () => {
    const body = `
      refused := fixture.send(t, request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
      assertNoNotificationSent(t, fixture.sender)
    `

    expect(effectEvidenceIn(body)).toContain('effect assertion')
  })

  test('does not count an asserted response or decoded problem as effect evidence', () => {
    const body = `
      response := fixture.send(t, request)
      if response.Code != http.StatusForbidden { t.Fatal("want refusal") }
      problem := decodeProblem(t, response)
      if problem.Type != "access_denied" { t.Fatal("wrong problem") }
    `

    expect(effectEvidenceIn(body)).toEqual([])
  })

  test('does not count names or comments as evidence', () => {
    const body = `
      // The refusal leaves the user unchanged and sends nothing.
      refused := send(t, http.MethodDelete, "/users/alice", request)
      if refused.Code != http.StatusForbidden { t.Fatal("want refusal") }
    `

    expect(effectEvidenceIn(body)).toEqual([])
  })
})

describe('reportSecurityTestGaps', () => {
  test('does not treat an HTTP test server response as a state change', () => {
    const report = reportSecurityTestGaps([
      {
        path: 'backend/example/client_test.go',
        name: 'TestClientError',
        body: `
          server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
            w.WriteHeader(http.StatusNotFound)
          }))
        `,
      },
    ])

    expect(report.refused).toBe(0)
  })

  test('excludes refusals returned by a remote SCIM server from local effect gaps', () => {
    const report = reportSecurityTestGaps([
      {
        path: 'backend/provisioning/client_scim/client_test.go',
        name: 'TestClientConflict',
        body: `
          _, _, err := client.CreateUser(ctx, rules, attributes)
          if !errors.Is(err, ErrConflict) { t.Fatal("want conflict") }
        `,
      },
    ])

    expect(report.refused).toBe(0)
    expect(report.gaps).toEqual([])
  })

  test('excludes token-client authentication probes that cannot mutate state', () => {
    const report = reportSecurityTestGaps([
      {
        path: 'backend/oauth2/handlers_http/client_auth_test.go',
        name: 'TestClientAuthenticationFailuresAreUniform',
        body: `
          req := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
          if response.Code != http.StatusUnauthorized { t.Fatal("want refusal") }
        `,
      },
    ])

    expect(report.refused).toBe(0)
    expect(report.gaps).toEqual([])
  })

  test('excludes a refused read whose setup creates the resource', () => {
    const report = reportSecurityTestGaps([
      {
        path: 'backend/sourcing/scim/handlers_http/scim_test.go',
        name: 'TestScimListUsersDateTimeFilterAndURNPrefix',
        body: `
          _, err := usecasesInst.CreateUser(ctx, tenantID, attributes)
          rec, _ := doScimGet(t, e, token, "/Users?filter=invalid")
          if rec.Code != http.StatusBadRequest { t.Fatal("want refusal") }
        `,
      },
    ])

    expect(report.refused).toBe(0)
    expect(report.gaps).toEqual([])
  })
})
