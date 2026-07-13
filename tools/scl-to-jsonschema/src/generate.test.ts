import { resolve } from 'node:path'
import Ajv2020 from 'ajv/dist/2020.js'
import addFormats from 'ajv-formats'
import { describe, expect, it } from 'bun:test'
import { loadSclBundle } from '../../scl-to-html/src/load.ts'
import type { SclDocument } from '../../scl-to-html/src/types.ts'
import { danglingRefs, generateModelSchemas, type JsonSchema } from './generate.ts'

const newAjv = () => {
  const ajv = new Ajv2020({ allErrors: true, strict: false })
  addFormats.default(ajv)
  return ajv
}

const defOf = (schema: JsonSchema, name: string): Record<string, unknown> => {
  const defs = schema.$defs as Record<string, unknown>
  const d = defs[name]
  if (!d || typeof d !== 'object') throw new Error(`missing $def: ${name}`)
  return d as Record<string, unknown>
}

const doc = (models: SclDocument['models']): SclDocument => ({
  system: 'demo',
  spec_version: '3.0',
  models,
})

describe('generateModelSchemas — unit', () => {
  it('maps an enum to a string enum', () => {
    const out = generateModelSchemas(doc({ Color: { kind: 'enum', values: ['Red', 'Blue'] } }))
    expect(defOf(out, 'Color')).toEqual({ type: 'string', enum: ['Red', 'Blue'] })
  })

  it('maps an entity, marking non-optional fields required and resolving refs', () => {
    const out = generateModelSchemas(
      doc({
        Color: { kind: 'enum', values: ['Red'] },
        Thing: {
          kind: 'entity',
          identity: 'id',
          fields: {
            id: { type: 'String', constraints: ['non_empty', { max_length: 10 }] },
            color: { type: 'Color' },
            note: { type: 'String', optional: true },
          },
        },
      }),
    )
    const thing = defOf(out, 'Thing')
    expect(thing.type).toBe('object')
    expect(thing.additionalProperties).toBe(false)
    expect(thing.required).toEqual(['id', 'color'])
    const props = thing.properties as Record<string, Record<string, unknown>>
    expect(props.id).toMatchObject({ type: 'string', minLength: 1, maxLength: 10 })
    expect(props.color).toEqual({ $ref: '#/$defs/Color' })
  })

  it('maps Name[] to an array of items', () => {
    const out = generateModelSchemas(
      doc({
        Tag: { kind: 'value_object', fields: { v: { type: 'String' } } },
        Bag: { kind: 'value_object', fields: { tags: { type: 'Tag[]' } } },
      }),
    )
    const bag = defOf(out, 'Bag')
    const props = bag.properties as Record<string, unknown>
    expect(props.tags).toEqual({ type: 'array', items: { $ref: '#/$defs/Tag' } })
  })

  it('treats Optional<T> as an optional field', () => {
    const out = generateModelSchemas(
      doc({
        Note: {
          kind: 'value_object',
          fields: { title: { type: 'String' }, body: { type: 'Optional<String>' } },
        },
      }),
    )
    expect(defOf(out, 'Note').required).toEqual(['title'])
  })

  it('maps SCL 3.0 field constraints to JSON Schema keywords', () => {
    const out = generateModelSchemas(
      doc({
        Item: {
          kind: 'value_object',
          fields: {
            count: { type: 'Integer', constraints: [{ minimum: 1 }, { maximum: 10 }] },
            code: { type: 'String', constraints: [{ pattern: '^[A-Z]+$' }] },
            tags: {
              type: 'List<String>',
              constraints: ['non_empty', 'unique', { max_length: 3 }],
            },
          },
        },
      }),
    )
    const props = defOf(out, 'Item').properties as Record<string, Record<string, unknown>>
    expect(props.count).toMatchObject({ type: 'integer', minimum: 1, maximum: 10 })
    expect(props.code).toMatchObject({ type: 'string', pattern: '^[A-Z]+$' })
    expect(props.tags).toMatchObject({
      type: 'array',
      minItems: 1,
      maxItems: 3,
      uniqueItems: true,
    })
  })

  it('enforces representable model CEL comparisons', () => {
    const out = generateModelSchemas(
      doc({
        Adult: {
          kind: 'value_object',
          fields: { age: { type: 'Integer' }, status: { type: 'String' } },
          constraints: ['age >= 18', 'status != "Deleted"'],
        },
      }),
    )
    const validate = newAjv().compile({ ...out, $ref: '#/$defs/Adult' })
    expect(validate({ age: 18, status: 'Active' })).toBe(true)
    expect(validate({ age: 17, status: 'Active' })).toBe(false)
    expect(validate({ age: 20, status: 'Deleted' })).toBe(false)
  })

  it('preserves unsupported model CEL explicitly instead of guessing', () => {
    const out = generateModelSchemas(
      doc({
        Window: {
          kind: 'value_object',
          fields: { start: { type: 'Integer' }, end: { type: 'Integer' } },
          constraints: ['start < end'],
        },
      }),
    )
    expect(defOf(out, 'Window')['x-scl-untranslated-constraints']).toEqual(['start < end'])
  })

  it('produces a schema with no dangling $defs references', () => {
    const out = generateModelSchemas(
      doc({
        A: { kind: 'value_object', fields: { b: { type: 'B' } } },
        B: { kind: 'value_object', fields: { x: { type: 'String' } } },
      }),
    )
    expect(danglingRefs(out)).toEqual([])
  })

  it('generated enum schema actually constrains values', () => {
    const out = generateModelSchemas(doc({ Color: { kind: 'enum', values: ['Red', 'Blue'] } }))
    const validate = newAjv().compile({ ...out, $ref: '#/$defs/Color' })
    expect(validate('Red')).toBe(true)
    expect(validate('Green')).toBe(false)
  })

  it('generated entity schema enforces required fields', () => {
    const out = generateModelSchemas(
      doc({
        Thing: { kind: 'entity', identity: 'id', fields: { id: { type: 'String' } } },
      }),
    )
    const validate = newAjv().compile({ ...out, $ref: '#/$defs/Thing' })
    expect(validate({ id: 'x' })).toBe(true)
    expect(validate({})).toBe(false)
  })
})

describe('generateModelSchemas — tool-spec conformance', () => {
  it('compiles as valid JSON Schema 2020-12 with no dangling refs', async () => {
    const sclPath = resolve(import.meta.dir, '../spec/scl.yaml')
    const bundle = await loadSclBundle(sclPath)
    const schema = generateModelSchemas(bundle)

    const defs = schema.$defs as Record<string, unknown>
    expect(Object.keys(defs).length).toBeGreaterThan(0)
    expect(danglingRefs(schema)).toEqual([])

    // Ajv compiling the whole document proves it is structurally a valid schema.
    expect(() => newAjv().compile(schema)).not.toThrow()
  })
})
