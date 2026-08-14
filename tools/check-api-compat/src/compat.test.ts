import { describe, expect, it } from 'bun:test'
import { compareOpenApi, type JsonSchema } from './compat.ts'

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
