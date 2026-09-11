import { describe, expect, it } from 'bun:test'
import { compareOpenApi, type JsonSchema } from './api-compat.ts'

const messages = (baseline: JsonSchema, current: JsonSchema) =>
  compareOpenApi(baseline, current).map((f) => `${f.operation}: ${f.message}`)

const withSchema = (schema: JsonSchema): JsonSchema => ({
  paths: {
    '/widgets': {
      get: {
        operationId: 'ListWidgets',
        responses: { '200': { content: { 'application/json': { schema } } } },
      },
    },
  },
})

describe('compareOpenApi — additive changes are not breaking', () => {
  it('reports nothing for two identical documents', () => {
    const doc = withSchema({
      type: 'object',
      properties: { id: { type: 'string' } },
      required: ['id'],
    })
    expect(compareOpenApi(doc, doc)).toEqual([])
  })

  it('does not flag a new path, a new operation, or a new optional field', () => {
    const baseline = {
      paths: {
        '/widgets': {
          get: {
            responses: {
              '200': {
                content: { 'application/json': { schema: { type: 'object', properties: {} } } },
              },
            },
          },
        },
      },
    }
    const current = {
      paths: {
        '/widgets': {
          get: {
            responses: {
              '200': {
                content: {
                  'application/json': {
                    schema: { type: 'object', properties: { note: { type: 'string' } } },
                  },
                },
              },
            },
          },
          post: { responses: { '201': {} } },
        },
        '/gizmos': { get: { responses: { '200': {} } } },
      },
    }
    expect(compareOpenApi(baseline, current)).toEqual([])
  })

  it('does not flag a new field the baseline never declared, even when it is required', () => {
    const baseline = withSchema({
      type: 'object',
      properties: { users: { type: 'integer' } },
      required: ['users'],
    })
    const current = withSchema({
      type: 'object',
      properties: { users: { type: 'integer' }, ssf_streams: { type: 'integer' } },
      required: ['users', 'ssf_streams'],
    })
    expect(compareOpenApi(baseline, current)).toEqual([])
  })

  it('does not flag a new error code added to an error response', () => {
    const baseline = withSchema({ oneOf: [{ $ref: '#/components/schemas/NotFound' }] })
    const current = withSchema({
      oneOf: [{ $ref: '#/components/schemas/NotFound' }, { $ref: '#/components/schemas/Conflict' }],
    })
    expect(compareOpenApi(baseline, current)).toEqual([])
  })

  it('does not flag a single response schema widened into an anyOf union', () => {
    const baseline = {
      ...withSchema({ $ref: '#/components/schemas/AccessDenied' }),
      components: {
        schemas: {
          AccessDenied: { type: 'object', properties: { type: { type: 'string' } } },
        },
      },
    }
    const current = {
      ...withSchema({
        anyOf: [
          { $ref: '#/components/schemas/AccessDenied' },
          { $ref: '#/components/schemas/InsufficientScope' },
        ],
      }),
      components: {
        schemas: {
          AccessDenied: { type: 'object', properties: { type: { type: 'string' } } },
          InsufficientScope: { type: 'object', properties: { type: { type: 'string' } } },
        },
      },
    }
    expect(compareOpenApi(baseline, current)).toEqual([])
  })
})

describe('compareOpenApi — breaking changes', () => {
  it('flags a removed path', () => {
    const baseline = { paths: { '/widgets': { get: { responses: {} } } } }
    const current = { paths: {} }
    expect(messages(baseline, current)).toEqual(['* /widgets: path removed'])
  })

  it('flags a removed operation on a surviving path', () => {
    const baseline = {
      paths: { '/widgets': { get: { responses: {} }, delete: { responses: {} } } },
    }
    const current = { paths: { '/widgets': { get: { responses: {} } } } }
    expect(messages(baseline, current)).toEqual(['DELETE /widgets: operation removed'])
  })

  it('flags a removed field', () => {
    const baseline = withSchema({
      type: 'object',
      properties: { id: { type: 'string' }, name: { type: 'string' } },
    })
    const current = withSchema({ type: 'object', properties: { id: { type: 'string' } } })
    expect(messages(baseline, current)).toEqual(["GET /widgets 200: field 'name' removed"])
  })

  it('flags a field that became required', () => {
    const baseline = withSchema({ type: 'object', properties: { name: { type: 'string' } } })
    const current = withSchema({
      type: 'object',
      properties: { name: { type: 'string' } },
      required: ['name'],
    })
    expect(messages(baseline, current)).toEqual(["GET /widgets 200: field 'name' became required"])
  })

  it('flags a field type change', () => {
    const baseline = withSchema({ type: 'object', properties: { count: { type: 'integer' } } })
    const current = withSchema({ type: 'object', properties: { count: { type: 'string' } } })
    expect(messages(baseline, current)).toEqual([
      "GET /widgets 200.count: type changed from 'integer' to 'string'",
    ])
  })

  it('flags a default value change', () => {
    const baseline = withSchema({
      type: 'object',
      properties: { limit: { type: 'integer', default: 10 } },
    })
    const current = withSchema({
      type: 'object',
      properties: { limit: { type: 'integer', default: 20 } },
    })
    expect(messages(baseline, current)).toEqual([
      'GET /widgets 200.limit: default value changed from 10 to 20',
    ])
  })

  it('flags a removed error code (oneOf ref)', () => {
    const baseline = withSchema({
      oneOf: [{ $ref: '#/components/schemas/NotFound' }, { $ref: '#/components/schemas/Conflict' }],
    })
    const current = withSchema({ oneOf: [{ $ref: '#/components/schemas/NotFound' }] })
    expect(messages(baseline, current)).toEqual(["GET /widgets 200: error code 'Conflict' removed"])
  })

  it('flags a removed field inside a $ref-resolved component schema', () => {
    const schemaRef = { $ref: '#/components/schemas/Widget' }
    const baseline = {
      ...withSchema(schemaRef),
      components: {
        schemas: {
          Widget: {
            type: 'object',
            properties: { id: { type: 'string' }, name: { type: 'string' } },
          },
        },
      },
    }
    const current = {
      ...withSchema(schemaRef),
      components: {
        schemas: { Widget: { type: 'object', properties: { id: { type: 'string' } } } },
      },
    }
    expect(messages(baseline, current)).toEqual(["GET /widgets 200: field 'name' removed"])
  })

  it('flags a removed request parameter and a newly required one', () => {
    const baseline = {
      paths: {
        '/widgets': {
          get: {
            parameters: [
              { name: 'q', in: 'query', required: false, schema: { type: 'string' } },
              { name: 'page', in: 'query', required: false, schema: { type: 'integer' } },
            ],
            responses: {},
          },
        },
      },
    }
    const current = {
      paths: {
        '/widgets': {
          get: {
            parameters: [
              { name: 'page', in: 'query', required: true, schema: { type: 'integer' } },
            ],
            responses: {},
          },
        },
      },
    }
    expect(messages(baseline, current)).toEqual([
      "GET /widgets: parameter 'q' removed",
      "GET /widgets: parameter 'page' became required",
    ])
  })

  it('flags a removed response status', () => {
    const baseline = { paths: { '/widgets': { get: { responses: { '200': {}, '404': {} } } } } }
    const current = { paths: { '/widgets': { get: { responses: { '200': {} } } } } }
    expect(messages(baseline, current)).toEqual(['GET /widgets 404: response status removed'])
  })

  it('flags a removed request body field', () => {
    const baseline = {
      paths: {
        '/widgets': {
          post: {
            requestBody: {
              content: {
                'application/json': {
                  schema: { type: 'object', properties: { name: { type: 'string' } } },
                },
              },
            },
            responses: {},
          },
        },
      },
    }
    const current = {
      paths: {
        '/widgets': {
          post: {
            requestBody: {
              content: { 'application/json': { schema: { type: 'object', properties: {} } } },
            },
            responses: {},
          },
        },
      },
    }
    expect(messages(baseline, current)).toEqual(["POST /widgets request: field 'name' removed"])
  })

  it('does not infinite-loop on a self-referential component schema', () => {
    const schemaRef = { $ref: '#/components/schemas/Node' }
    const baseline = {
      ...withSchema(schemaRef),
      components: {
        schemas: {
          Node: { type: 'object', properties: { child: { $ref: '#/components/schemas/Node' } } },
        },
      },
    }
    expect(() => compareOpenApi(baseline, baseline)).not.toThrow()
    expect(compareOpenApi(baseline, baseline)).toEqual([])
  })
})

// OpenAPI 3.1 は JSON Schema 2020-12 なので、`$ref` と並んで書かれた keyword は
// 適用される。無視されたのは 3.0 までの規則である。生成器はこの形を実際に出力し、
// `description` と `default` を `$ref` の傍らに置く。
describe('compareOpenApi — keywords written beside a $ref', () => {
  const components = { schemas: { Kind: { type: 'string', enum: ['a', 'b'] } } }
  const withProperty = (property: JsonSchema): JsonSchema => ({
    ...withSchema({ type: 'object', properties: { kind: property } }),
    components,
  })

  it('reports a default that changed beside a $ref', () => {
    const baseline = withProperty({ $ref: '#/components/schemas/Kind', default: 'a' })
    const current = withProperty({ $ref: '#/components/schemas/Kind', default: 'b' })
    expect(messages(baseline, current)).toEqual([
      'GET /widgets 200.kind: default value changed from "a" to "b"',
    ])
  })

  it('does not flag moving an inline schema to a $ref that keeps the same default', () => {
    const baseline = withProperty({ type: 'string', enum: ['a', 'b'], default: 'a' })
    const current = withProperty({ $ref: '#/components/schemas/Kind', default: 'a' })
    expect(messages(baseline, current)).toEqual([])
  })

  it('keeps the resolved schema required fields when the reference adds its own', () => {
    const target = {
      schemas: {
        Widget: { type: 'object', properties: { id: {}, note: {} }, required: ['id'] },
      },
    }
    const baseline = {
      ...withSchema({ $ref: '#/components/schemas/Widget' }),
      components: target,
    }
    const current = {
      ...withSchema({ $ref: '#/components/schemas/Widget', required: ['note'] }),
      components: target,
    }
    // `id` は解決先が要求したままで、`note` が新しく要求される。傍らの required で
    // 解決先の required を置き換えると、`id` が要求されなくなったことを見落とす。
    expect(messages(baseline, current)).toEqual(["GET /widgets 200: field 'note' became required"])
  })

  it('keeps a resolved property the reference does not restate', () => {
    const target = {
      schemas: { Widget: { type: 'object', properties: { id: {}, note: {} } } },
    }
    const baseline = {
      ...withSchema({ $ref: '#/components/schemas/Widget' }),
      components: target,
    }
    // 傍らの properties は `id` だけを書き直す。解決先の `note` は残っているので、
    // 傍らで置き換える読み方が作る「note が消えた」は誤報である。
    const current = {
      ...withSchema({ $ref: '#/components/schemas/Widget', properties: { id: {} } }),
      components: target,
    }
    expect(messages(baseline, current)).toEqual([])
  })

  it('keeps a field the resolved schema already required when the baseline names another beside the $ref', () => {
    const target = {
      schemas: {
        Widget: { type: 'object', properties: { id: {}, note: {} }, required: ['id'] },
      },
    }
    // baseline は解決先の required に `note` を足しており、現行はそれをやめた。
    // 要求が外れることは破壊的変更ではないので報告は出ない。傍らの required で
    // 解決先を置き換えると、baseline が `id` を要求していなかったことになり、
    // 現行の `id` が「新しく必須になった」という誤報になる。
    const baseline = {
      ...withSchema({ $ref: '#/components/schemas/Widget', required: ['note'] }),
      components: target,
    }
    const current = {
      ...withSchema({ $ref: '#/components/schemas/Widget' }),
      components: target,
    }
    expect(messages(baseline, current)).toEqual([])
  })

  it('reports a property the reference narrows beside the $ref', () => {
    const target = {
      schemas: { Widget: { type: 'object', properties: { id: { type: 'string' } } } },
    }
    const beside = (property: JsonSchema): JsonSchema => ({
      ...withSchema({ $ref: '#/components/schemas/Widget', properties: { id: property } }),
      components: target,
    })
    // 解決先は両方とも同じなので、傍らの properties を捨てる読み方では差が消える。
    expect(messages(beside({ type: 'string' }), beside({ type: 'integer' }))).toEqual([
      "GET /widgets 200.id: type changed from 'string' to 'integer'",
    ])
  })
})
