import { resolve } from 'node:path'
import Ajv2020 from 'ajv/dist/2020.js'
import addFormats from 'ajv-formats'
import { describe, expect, it } from 'bun:test'
import { loadSclBundle } from '../../scl-to-html/src/load.ts'
import type { Authorization, SclDocument } from '../../scl-to-html/src/types.ts'
import {
  collectRefNames,
  type JsonSchema,
  missingRefs,
} from '../../scl-to-jsonschema/src/generate.ts'
import { generateOpenApi } from './openapi.ts'

const newAjv = () => {
  const ajv = new Ajv2020({ allErrors: true, strict: false })
  addFormats.default(ajv)
  return ajv
}

const op = (doc: JsonSchema, path: string, method: string): Record<string, unknown> => {
  const paths = doc.paths as Record<string, Record<string, unknown>>
  const item = paths[path]
  if (!item) throw new Error(`no path ${path}`)
  const o = item[method]
  if (!o || typeof o !== 'object') throw new Error(`no ${method} ${path}`)
  return o as Record<string, unknown>
}

const doc = (
  models: SclDocument['models'],
  interfaces: SclDocument['interfaces'],
  authorization?: Authorization,
): SclDocument => ({ system: 'demo', spec_version: '3.0', models, interfaces, authorization })

describe('generateOpenApi — unit', () => {
  it('turns an http interface into an operation with a json request body', () => {
    const out = generateOpenApi(
      doc(
        { Req: { kind: 'value_object', fields: { a: { type: 'String' } } } },
        {
          DoThing: {
            description: 'Do a thing.\nmore detail',
            input: { request: { type: 'Req' } },
            bindings: [{ kind: 'http', method: 'POST', path: '/things', request_form: 'body' }],
          },
        },
      ),
    )
    const o = op(out, '/things', 'post')
    expect(o.operationId).toBe('DoThing')
    expect(o.summary).toBe('Do a thing.')
    const body = o.requestBody as { content: Record<string, { schema: Record<string, unknown> }> }
    const schema = body.content['application/json']?.schema as Record<string, unknown>
    // request schema references the model under components/schemas
    const props = schema.properties as Record<string, unknown>
    expect(props.request).toEqual({ $ref: '#/components/schemas/Req' })
  })

  it('maps query request_form to query parameters and path tokens to path params', () => {
    const out = generateOpenApi(
      doc(undefined, {
        Get: {
          input: {
            id: { type: 'String' },
            q: { type: 'String' },
            opt: { type: 'String', optional: true },
          },
          bindings: [{ kind: 'http', method: 'GET', path: '/t/{id}', request_form: 'query' }],
        },
      }),
    )
    const o = op(out, '/t/{id}', 'get')
    const params = o.parameters as Array<Record<string, unknown>>
    expect(params).toContainEqual({
      name: 'id',
      in: 'path',
      required: true,
      schema: { type: 'string' },
    })
    expect(params).not.toContainEqual({
      name: 'id',
      in: 'query',
      required: true,
      schema: { type: 'string' },
    })
    expect(params).toContainEqual({
      name: 'q',
      in: 'query',
      required: true,
      schema: { type: 'string' },
    })
    expect(params).toContainEqual({
      name: 'opt',
      in: 'query',
      required: false,
      schema: { type: 'string' },
    })
  })

  it('emits success status codes and groups errors by their model status', () => {
    const out = generateOpenApi(
      doc(
        {
          E: { kind: 'error', status: 400 },
          Resp: { kind: 'value_object', fields: { x: { type: 'String' } } },
        },
        {
          Op: {
            output: { response: { type: 'Resp' } },
            errors: ['E'],
            bindings: [
              {
                kind: 'http',
                method: 'POST',
                path: '/op',
                successful_status_codes: ['201'],
              },
            ],
          },
        },
      ),
    )
    const o = op(out, '/op', 'post')
    const responses = o.responses as Record<string, Record<string, unknown>>
    expect(responses['201']).toBeDefined()
    expect(responses.default).toBeUndefined()
    const errorResponse = responses['400'] as { description: string; content: JsonSchema }
    expect(errorResponse.description).toContain('E')
    expect(errorResponse.content['application/problem+json']).toBeDefined()
  })

  it('groups multiple errors that share a status under one response', () => {
    const out = generateOpenApi(
      doc(
        { A: { kind: 'error', status: 422 }, B: { kind: 'error', status: 422 } },
        {
          Op: {
            errors: ['A', 'B'],
            bindings: [{ kind: 'http', method: 'POST', path: '/op' }],
          },
        },
      ),
    )
    const o = op(out, '/op', 'post')
    const responses = o.responses as Record<string, Record<string, unknown>>
    const errorResponse = responses['422'] as {
      description: string
      content: Record<string, { schema: Record<string, unknown> }>
    }
    expect(errorResponse.description).toContain('A')
    expect(errorResponse.description).toContain('B')
    const schema = errorResponse.content['application/problem+json']?.schema
    expect(schema?.oneOf).toEqual([
      { $ref: '#/components/schemas/A' },
      { $ref: '#/components/schemas/B' },
    ])
  })

  it('uses the binding error_format to pick the error content-type', () => {
    const cases: Array<[string, string]> = [
      ['oauth2', 'application/json'],
      ['scim', 'application/scim+json'],
      ['set_delivery', 'application/json'],
    ]
    for (const [errorFormat, contentType] of cases) {
      const out = generateOpenApi(
        doc(
          { E: { kind: 'error', status: 400 } },
          {
            Op: {
              errors: ['E'],
              bindings: [{ kind: 'http', method: 'POST', path: '/op', error_format: errorFormat }],
            },
          },
        ),
      )
      const o = op(out, '/op', 'post')
      const responses = o.responses as Record<string, Record<string, unknown>>
      const errorResponse = responses['400'] as { content: Record<string, unknown> }
      expect(Object.keys(errorResponse.content)).toEqual([contentType])
    }
  })

  it('emits public and protected security metadata with local contracts', () => {
    const out = generateOpenApi(
      doc(
        {
          User: { kind: 'entity', identity: 'id', fields: { id: { type: 'UUID' } } },
          Tenant: { kind: 'entity', identity: 'id', fields: { id: { type: 'UUID' } } },
        },
        {
          Health: {
            access: 'public',
            bindings: [{ kind: 'http', method: 'GET', path: '/health' }],
          },
          UpdateTenant: {
            input: { id: { type: 'UUID' } },
            requires: ['input.id != ""'],
            ensures: ['response.status == 200'],
            access: {
              policies: ['TenantMember'],
              resource: { type: 'Tenant', id: 'input.id' },
            },
            bindings: [{ kind: 'http', method: 'PATCH', path: '/tenants/{id}' }],
          },
        },
        {
          principals: { Member: { type: 'User', matches: ['principal.id != ""'] } },
          policies: { TenantMember: { effect: 'permit', principal: 'Member' } },
        },
      ),
    )

    expect(op(out, '/health', 'get').security).toEqual([])
    const update = op(out, '/tenants/{id}', 'patch')
    expect(update.security).toEqual([{ SclBearer: [] }])
    expect(update['x-scl-access']).toEqual({
      policies: ['TenantMember'],
      resource: { type: 'Tenant', id: 'input.id' },
    })
    expect(update['x-scl-requires']).toEqual(['input.id != ""'])
    expect(update['x-scl-ensures']).toEqual(['response.status == 200'])
    expect(update.description).toContain('Requires: input.id != ""')

    const components = out.components as Record<string, unknown>
    expect(components.securitySchemes).toEqual({
      SclBearer: { type: 'http', scheme: 'bearer', bearerFormat: 'JWT' },
    })
    expect(out['x-scl-authorization']).toMatchObject({
      groups: [{ name: 'policy:TenantMember' }],
    })
  })

  it('does not expose an invalid internal http interface', () => {
    const out = generateOpenApi(
      doc(undefined, {
        InternalOnly: {
          access: 'internal',
          bindings: [{ kind: 'http', method: 'POST', path: '/internal' }],
        },
      }),
    )
    expect((out.paths as Record<string, unknown>)['/internal']).toBeUndefined()
  })

  it('merges different methods that share one path item', () => {
    const out = generateOpenApi(
      doc(undefined, {
        ListResources: {
          bindings: [{ kind: 'http', method: 'GET', path: '/resources' }],
        },
        CreateResource: {
          bindings: [{ kind: 'http', method: 'POST', path: '/resources' }],
        },
      }),
    )

    expect(op(out, '/resources', 'get').operationId).toBe('ListResources')
    expect(op(out, '/resources', 'post').operationId).toBe('CreateResource')
  })

  it('rejects duplicate method and path bindings', () => {
    expect(() =>
      generateOpenApi(
        doc(undefined, {
          First: {
            bindings: [{ kind: 'http', method: 'GET', path: '/resources' }],
          },
          Second: {
            bindings: [{ kind: 'http', method: 'GET', path: '/resources' }],
          },
        }),
      ),
    ).toThrow('duplicate HTTP binding GET /resources: First, Second')
  })

  // ADR-156 / wi-297 T006: stable/beta interfaces under a versioned prefix
  // (/api/admin, /api/account) gain a "v1" alias path mirroring the runtime
  // RegisterVersionAliases hook (backend/shared/http/support_http).
  it('mirrors stable interfaces under /api/admin and /api/account at their v1 alias path', () => {
    const out = generateOpenApi(
      doc(undefined, {
        ListWidgets: {
          access: 'public',
          stability: 'stable',
          bindings: [{ kind: 'http', method: 'GET', path: '/api/admin/widgets' }],
        },
        GetProfile: {
          access: 'public',
          stability: 'stable',
          bindings: [{ kind: 'http', method: 'GET', path: '/api/account/profile' }],
        },
      }),
    )
    expect(op(out, '/api/admin/widgets', 'get').operationId).toBe('ListWidgets')
    expect(op(out, '/api/admin/v1/widgets', 'get').operationId).toBe('ListWidgets')
    expect(op(out, '/api/account/profile', 'get').operationId).toBe('GetProfile')
    expect(op(out, '/api/account/v1/profile', 'get').operationId).toBe('GetProfile')
  })

  it('aliases internal-stability interfaces too, matching the path-only runtime hook', () => {
    const out = generateOpenApi(
      doc(undefined, {
        AdminConsoleOnly: {
          access: {
            policies: ['TenantAdministrator'],
            resource: { type: 'Tenant', id: 'context.tenant_id' },
          },
          stability: 'internal',
          bindings: [{ kind: 'http', method: 'GET', path: '/api/admin/console-only' }],
        },
      }),
    )
    expect(op(out, '/api/admin/console-only', 'get').operationId).toBe('AdminConsoleOnly')
    expect(op(out, '/api/admin/v1/console-only', 'get').operationId).toBe('AdminConsoleOnly')
  })

  it('does not alias paths outside a versioned prefix', () => {
    const out = generateOpenApi(
      doc(undefined, {
        Login: {
          access: 'public',
          stability: 'internal',
          bindings: [{ kind: 'http', method: 'POST', path: '/api/auth/login' }],
        },
      }),
    )
    expect((out.paths as Record<string, unknown>)['/api/auth/v1/login']).toBeUndefined()
  })

  it('reflects stability and deprecation metadata on the operation', () => {
    const out = generateOpenApi(
      doc(undefined, {
        OldWidgets: {
          access: 'public',
          stability: 'stable',
          deprecated_since: '2026-01-01',
          sunset_at: '2027-01-01',
          successor: 'NewWidgets',
          bindings: [{ kind: 'http', method: 'GET', path: '/api/admin/old-widgets' }],
        },
      }),
    )
    const operation = op(out, '/api/admin/old-widgets', 'get')
    expect(operation.deprecated).toBe(true)
    expect(operation['x-scl-stability']).toBe('stable')
    expect(operation['x-scl-deprecated-since']).toBe('2026-01-01')
    expect(operation['x-scl-sunset-at']).toBe('2027-01-01')
    expect(operation['x-scl-successor']).toBe('NewWidgets')

    const alias = op(out, '/api/admin/v1/old-widgets', 'get')
    expect(alias.deprecated).toBe(true)
  })
})

describe('generateOpenApi — tool-spec conformance', () => {
  it('produces a 3.1 document whose refs all resolve to components/schemas', async () => {
    const sclPath = resolve(import.meta.dir, '../spec/scl.yaml')
    const bundle = await loadSclBundle(sclPath)
    const out = generateOpenApi(bundle)

    expect(out.openapi).toBe('3.1.0')
    const paths = out.paths as Record<string, unknown>
    expect(paths).toBeDefined()

    const components = out.components as { schemas: Record<string, unknown> }
    const known = new Set(Object.keys(components.schemas))
    // Every $ref in the whole document resolves, and none still point at $defs.
    expect(missingRefs(out, known, '#/components/schemas/')).toEqual([])
    expect(collectRefNames(out, '#/$defs/')).toEqual([])

    // Each component schema is a valid JSON Schema 2020-12 (OpenAPI 3.1 uses it).
    const ajv = newAjv()
    expect(() => ajv.compile({ $defs: components.schemas })).not.toThrow()
  })
})
