import { describe, expect, it } from 'bun:test'
import {
  collectGoHandlers,
  collectGoStructs,
  collectRoutes,
  diffContract,
  type OpenAPIDocument,
} from './contract-drift.ts'

/**
 * The chain these tests exercise is the one wi-382 walked by hand: operationId
 * to route to handler to the struct the handler decodes into. Each step is
 * tested for the shape it actually has to survive in this repository, and the
 * unresolved cases are asserted as loudly as the findings — a checker that
 * silently drops what it cannot follow is worse than none.
 */

const route = (source: string) => collectRoutes([{ path: 'routes.go', source }])

describe('collectRoutes', () => {
  it('normalizes echo :param to the OpenAPI {param} form', () => {
    const routes = route(`
func Register(g *echo.Group, d Deps) {
	g.POST("/api/admin/v1/applications/:id/provisioning", d.handleRegisterConnection)
}`)
    expect(routes.get('POST /api/admin/v1/applications/{id}/provisioning')).toBe(
      'handleRegisterConnection',
    )
  })

  it('reads a handler registered without a receiver', () => {
    const routes = route(`g.GET("/healthz", handleHealth)`)
    expect(routes.get('GET /healthz')).toBe('handleHealth')
  })

  it('follows a route registered through a delegating closure', () => {
    // The Authentication package registers most of its account routes this way,
    // and reading only the direct form left 109 operations unresolved.
    const routes = route(
      `g.GET("/api/account/v1/consents", func(c *echo.Context) error { return handleListAccountConsents(d, c) })`,
    )
    expect(routes.get('GET /api/account/v1/consents')).toBe('handleListAccountConsents')
  })

  it('keeps the registered parameter name so a caller can compare it', () => {
    const routes = route(`g.GET("/api/admin/v1/tenants/:target_tenant_id", d.handleGetTenant)`)
    expect(routes.get('GET /api/admin/v1/tenants/{target_tenant_id}')).toBe('handleGetTenant')
  })

  it('keeps every HTTP method apart', () => {
    const routes = route(`
	g.GET("/a", d.get)
	g.DELETE("/a", d.remove)`)
    expect(routes.get('GET /a')).toBe('get')
    expect(routes.get('DELETE /a')).toBe('remove')
  })
})

describe('collectGoHandlers', () => {
  it('cuts the body at the matching brace, not the first one', () => {
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	if x {
		return nil
	}
	var req createRequest
	return support.DecodeJSON(c.Request(), &req)
}

func (d Deps) handleB(c *echo.Context) error {
	var other otherRequest
	_ = other
	return nil
}`,
      },
    ])
    expect(handlers.get('handleA')?.body).toContain('createRequest')
    // handleB's local must not leak into handleA's body.
    expect(handlers.get('handleA')?.body).not.toContain('otherRequest')
    expect(handlers.get('handleB')?.body).toContain('otherRequest')
  })

  it('reads a handler that takes its dependencies as a parameter', () => {
    // `func handleX(d Deps, c *echo.Context) error` is the form the delegating
    // closures above call.
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func handleListAccountConsents(d Deps, c *echo.Context) error {
	var req listRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil
}`,
      },
    ])
    expect(handlers.get('handleListAccountConsents')?.requestType).toBe('listRequest')
  })

  it('resolves the decoded request type through the local declaration', () => {
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	var req registerConnectionRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return err
	}
	return nil
}`,
      },
    ])
    expect(handlers.get('handleA')?.requestType).toBe('registerConnectionRequest')
  })

  it('resolves c.Bind as well as DecodeJSON', () => {
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	var input updateRequest
	if err := c.Bind(&input); err != nil {
		return err
	}
	return nil
}`,
      },
    ])
    expect(handlers.get('handleA')?.requestType).toBe('updateRequest')
  })

  it('reports no request type when the handler decodes nothing', () => {
    const handlers = collectGoHandlers([
      { path: 'h.go', source: `func (d Deps) handleA(c *echo.Context) error { return nil }` },
    ])
    expect(handlers.get('handleA')?.requestType).toBeUndefined()
  })

  it('reads the response keys of a map literal written in place', () => {
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"id": id, "status": status,
	})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseKeys).toEqual(['id', 'status'])
  })

  it('reads only the first level of a nested map literal', () => {
    // The contract compares envelopes, so a nested key is not a response key.
    // Reading them all reported five spurious extras on CheckAccess alone.
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"result": map[string]any{
		"permitted":     result.Permitted,
		"model_version": result.ModelVersion,
	}})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseKeys).toEqual(['result'])
  })

  it('reads a map literal of any value type, not only map[string]any', () => {
    // Reading only map[string]any left 30-odd operations unchecked for no reason
    // other than the element type the handler happened to write.
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"csrf_token": csrf})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseKeys).toEqual(['csrf_token'])
  })

  it('does not mistake a quoted string value for a key', () => {
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"status": "queued"})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseKeys).toEqual(['status'])
  })

  it('reads the response type of a struct literal written in place', () => {
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	return c.JSON(http.StatusOK, listResponse{Items: items})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseType).toBe('listResponse')
  })

  it('reads the success write, not whichever write comes first', () => {
    // handleReadyz writes 503 twice before its 200. Taking the first match
    // compared the contract's success body against an error body and reported a
    // difference that was not there.
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	if draining {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": s, "dependencies": d})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseKeys?.sort()).toEqual(['dependencies', 'status'])
  })

  it('stays unresolved when two success writes disagree about the shape', () => {
    // Guessing between them is how a checker starts reporting differences that
    // depend on which branch it happened to read.
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	if verbose {
		return c.JSON(http.StatusOK, map[string]any{"status": s, "dependencies": d})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": s})
}`,
      },
    ])
    expect(handlers.get('handleA')?.responseKeys).toBeUndefined()
  })

  it('leaves the response unresolved when a variable is returned', () => {
    // This is the honest half: `conn` needs type inference the regex does not do.
    const handlers = collectGoHandlers([
      {
        path: 'h.go',
        source: `
func (d Deps) handleA(c *echo.Context) error {
	return support.NoStoreJSON(c, http.StatusCreated, conn)
}`,
      },
    ])
    const handler = handlers.get('handleA')
    expect(handler?.responseKeys).toBeUndefined()
    expect(handler?.responseType).toBeUndefined()
  })
})

describe('collectGoStructs', () => {
  it('collects json tag names and drops omitempty and the ignore marker', () => {
    const structs = collectGoStructs([
      {
        path: 's.go',
        source: `
type registerConnectionRequest struct {
	BaseURL    string            \`json:"base_url"\`
	Credential credentialRequest \`json:"credential,omitempty"\`
	internal   string            \`json:"-"\`
	NoTag      string
}`,
      },
    ])
    expect(structs.get('registerConnectionRequest')).toEqual(['base_url', 'credential'])
  })

  it('keeps two structs in one file apart', () => {
    const structs = collectGoStructs([
      {
        path: 's.go',
        source: `
type a struct {
	X string \`json:"x"\`
}

type b struct {
	Y string \`json:"y"\`
}`,
      },
    ])
    expect(structs.get('a')).toEqual(['x'])
    expect(structs.get('b')).toEqual(['y'])
  })
})

/** A minimal compiled-OpenAPI shape carrying one operation. */
function documentWith(operation: Record<string, unknown>): OpenAPIDocument {
  return {
    paths: { '/api/admin/v1/things/{id}': { post: { operationId: 'CreateThing', ...operation } } },
    components: { schemas: {} },
  } as OpenAPIDocument
}

const goSources = (handlerBody: string, structs: string) => ({
  routes: collectRoutes([
    { path: 'routes.go', source: `g.POST("/api/admin/v1/things/:id", d.handleCreate)` },
  ]),
  handlers: [
    { path: 'h.go', source: `func (d Deps) handleCreate(c *echo.Context) error {${handlerBody}}` },
  ],
  structs: [{ path: 's.go', source: structs }],
})

describe('diffContract', () => {
  const requestBody = (properties: Record<string, unknown>) => ({
    requestBody: {
      content: { 'application/json': { schema: { type: 'object', properties } } },
    },
  })

  it('reports a request property the Go struct does not decode', () => {
    // wi-381's defect: the contract declares fields the handler never reads.
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name string \`json:"name"\`
}`,
    )
    const result = diffContract(
      documentWith(requestBody({ name: {}, description: {} })),
      go.routes,
      [...go.handlers, ...go.structs],
    )
    const messages = result.findings.map((f) => f.message)
    expect(messages.some((m) => m.includes('D1') && m.includes('description'))).toBe(true)
  })

  it('reports a Go field the contract does not declare', () => {
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name        string \`json:"name"\`
	Description string \`json:"description"\`
}`,
    )
    const result = diffContract(documentWith(requestBody({ name: {} })), go.routes, [
      ...go.handlers,
      ...go.structs,
    ])
    expect(result.findings.some((f) => f.message.includes('description'))).toBe(true)
  })

  it('passes when the property sets match', () => {
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name string \`json:"name"\`
}`,
    )
    const result = diffContract(documentWith(requestBody({ name: {} })), go.routes, [
      ...go.handlers,
      ...go.structs,
    ])
    expect(result.findings).toEqual([])
    expect(result.unresolved).toEqual([])
  })

  it('reports a path parameter also declared as a request body property', () => {
    // wi-382's other defect class: the id is in the path and in the body.
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	ID   string \`json:"id"\`
	Name string \`json:"name"\`
}`,
    )
    const document = documentWith({
      parameters: [{ name: 'id', in: 'path' }],
      ...requestBody({ id: {}, name: {} }),
    })
    const result = diffContract(document, go.routes, [...go.handlers, ...go.structs])
    expect(result.findings.some((f) => f.message.includes('D2') && f.message.includes('id'))).toBe(
      true,
    )
  })

  it('reports a response envelope the handler does not produce', () => {
    // wi-382's fictitious wrapper: the contract says {"response": {...}}, the
    // handler writes the resource itself.
    const go = goSources(
      `
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"id": id})`,
      '',
    )
    const document = documentWith({
      responses: {
        '200': {
          content: {
            'application/json': {
              schema: { type: 'object', properties: { response: {} } },
            },
          },
        },
      },
    })
    const result = diffContract(document, go.routes, [...go.handlers, ...go.structs])
    expect(
      result.findings.some((f) => f.message.includes('D3') && f.message.includes('response')),
    ).toBe(true)
  })

  it('matches a route whose parameter is spelled differently, when the shape is unique', () => {
    // The contract says {tenant_id}; the code registers :target_tenant_id. The
    // bodies are still comparable, and refusing to compare them would hide a
    // real body drift behind a naming difference.
    const routes = collectRoutes([
      { path: 'routes.go', source: `g.POST("/api/admin/v1/things/:thing_id", d.handleCreate)` },
    ])
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name string \`json:"name"\`
}`,
    )
    const result = diffContract(documentWith(requestBody({ name: {}, extra: {} })), routes, [
      ...go.handlers,
      ...go.structs,
    ])
    expect(result.unresolved).toEqual([])
    expect(result.findings.some((f) => f.message.includes('extra'))).toBe(true)
  })

  it('stays unresolved when two routes share the same shape', () => {
    // Falling back to shape is only safe while it names one handler. Comparing
    // against the wrong one is the silent-wrong-answer this check must not give.
    const routes = collectRoutes([
      {
        path: 'routes.go',
        source: `
	g.POST("/api/admin/v1/things/:a", d.handleCreate)
	g.POST("/api/admin/v1/things/:b", d.handleOther)`,
      },
    ])
    const result = diffContract(documentWith(requestBody({ name: {} })), routes, [])
    expect(result.unresolved[0]?.reason).toBe('route-not-found')
  })

  it('keys a finding by rule and operation so the baseline survives a rename', () => {
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name string \`json:"name"\`
}`,
    )
    const result = diffContract(documentWith(requestBody({ name: {}, extra: {} })), go.routes, [
      ...go.handlers,
      ...go.structs,
    ])
    expect(result.findings[0]?.key).toBe('D1 CreateThing')
  })

  it('compares a nested object property against the nested Go struct', () => {
    // wi-381's defect lived in UserAttributeDef, which is nested inside another
    // model rather than being a request body itself. A check that only reads the
    // first level would not fail if that defect came back — which is the one
    // thing this work item's Verification asks for by name.
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name       string            \`json:"name"\`
	Credential credentialRequest \`json:"credential"\`
}

type credentialRequest struct {
	AuthMethod string \`json:"auth_method"\`
}`,
    )
    const document = {
      paths: {
        '/api/admin/v1/things/{id}': {
          post: {
            operationId: 'CreateThing',
            requestBody: {
              content: {
                'application/json': {
                  schema: {
                    type: 'object',
                    properties: {
                      name: {},
                      credential: { $ref: '#/components/schemas/Credential' },
                    },
                  },
                },
              },
            },
          },
        },
      },
      components: {
        schemas: {
          Credential: {
            type: 'object',
            properties: { auth_method: {}, bearer_token: {} },
          },
        },
      },
    } as unknown as OpenAPIDocument
    const result = diffContract(document, go.routes, [...go.handlers, ...go.structs])
    expect(
      result.findings.some(
        (f) => f.message.includes('credential') && f.message.includes('bearer_token'),
      ),
    ).toBe(true)
  })

  it('does not recurse into a nested property the Go struct does not name', () => {
    // The missing property is already reported at the level above; descending
    // into it would report the same drift twice.
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name string \`json:"name"\`
}`,
    )
    const document = {
      paths: {
        '/api/admin/v1/things/{id}': {
          post: {
            operationId: 'CreateThing',
            requestBody: {
              content: {
                'application/json': {
                  schema: {
                    type: 'object',
                    properties: { name: {}, nested: { type: 'object', properties: { a: {} } } },
                  },
                },
              },
            },
          },
        },
      },
      components: { schemas: {} },
    } as unknown as OpenAPIDocument
    const result = diffContract(document, go.routes, [...go.handlers, ...go.structs])
    expect(result.findings).toHaveLength(1)
    expect(result.findings[0]?.message).toContain('nested')
  })

  it('records an operation whose route is not registered as unresolved', () => {
    const result = diffContract(documentWith(requestBody({ name: {} })), new Map(), [])
    expect(result.findings).toEqual([])
    expect(result.unresolved).toHaveLength(1)
    expect(result.unresolved[0]?.reason).toBe('route-not-found')
  })

  it('records an unfollowable decode target as unresolved rather than passing it', () => {
    // The rule the Risk Notes ask for: what the checker cannot follow must not
    // be counted as agreement.
    const go = goSources(
      `
	_ = support.DecodeJSON(c.Request(), &d.buffer)
	return nil`,
      '',
    )
    const result = diffContract(documentWith(requestBody({ name: {} })), go.routes, [
      ...go.handlers,
      ...go.structs,
    ])
    expect(result.findings).toEqual([])
    expect(result.unresolved[0]?.reason).toBe('request-type-not-found')
  })

  it('does not fault an operation that declares no request body', () => {
    const go = goSources('\n\treturn nil', '')
    const result = diffContract(documentWith({}), go.routes, [...go.handlers, ...go.structs])
    expect(result.findings).toEqual([])
    expect(result.unresolved).toEqual([])
  })

  it('does not fault a nested response key against a first-level contract key', () => {
    // The regression that produced five spurious extras on CheckAccess: the
    // handler writes {"result": {...}} and the contract declares `result`.
    const go = goSources(
      `
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"result": map[string]any{
		"permitted": p, "model_version": v,
	}})`,
      '',
    )
    const document = documentWith({
      responses: {
        '200': {
          content: {
            'application/json': { schema: { type: 'object', properties: { result: {} } } },
          },
        },
      },
    })
    const result = diffContract(document, go.routes, [...go.handlers, ...go.structs])
    expect(result.findings).toEqual([])
  })

  it('resolves a $ref request schema through components', () => {
    const document = {
      paths: {
        '/api/admin/v1/things/{id}': {
          post: {
            operationId: 'CreateThing',
            requestBody: {
              content: {
                'application/json': { schema: { $ref: '#/components/schemas/CreateThingBody' } },
              },
            },
          },
        },
      },
      components: {
        schemas: { CreateThingBody: { type: 'object', properties: { name: {}, extra: {} } } },
      },
    } as unknown as OpenAPIDocument
    const go = goSources(
      `
	var req createThingRequest
	_ = support.DecodeJSON(c.Request(), &req)
	return nil`,
      `type createThingRequest struct {
	Name string \`json:"name"\`
}`,
    )
    const result = diffContract(document, go.routes, [...go.handlers, ...go.structs])
    expect(result.findings.some((f) => f.message.includes('extra'))).toBe(true)
  })
})
