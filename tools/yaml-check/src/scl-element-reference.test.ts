import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import {
  buildSclWorkspaceIndex,
  canonicalSclElementReference,
  normalizeSclElementReference,
  resolveSclElementReference,
  sclElementAnchor,
  type SclElementReference,
} from './scl-element-reference.ts'

const fixtureDir = resolve(import.meta.dir, 'fixtures/scl-element-references')

async function yaml(path: string): Promise<unknown> {
  return Bun.YAML.parse(await Bun.file(resolve(fixtureDir, path)).text())
}

async function fixtureIndex() {
  const built = buildSclWorkspaceIndex(await yaml('root.yaml'), {
    Identity: await yaml('contexts/identity.yaml'),
    System: await yaml('contexts/system.yaml'),
  })
  if (!built.ok) throw new Error(built.errors.map((item) => item.message).join('; '))
  return built.index
}

describe('SCL element reference', () => {
  it('normalizes the two closed authored shapes', () => {
    expect(
      normalizeSclElementReference({ context: 'Identity', kind: 'model', element: 'User' }),
    ).toEqual({
      ok: true,
      reference: { context: 'Identity', kind: 'model', element: 'User' },
    })
    expect(
      normalizeSclElementReference({
        context: 'System',
        kind: 'standard_requirement',
        standard: 'ExampleStandard',
        requirement: 'REQ-1',
      }),
    ).toEqual({
      ok: true,
      reference: {
        context: 'System',
        kind: 'standard_requirement',
        standard: 'ExampleStandard',
        requirement: 'REQ-1',
      },
    })
  })

  it('rejects unknown kinds, missing fields, and position-based fields', () => {
    expect(
      normalizeSclElementReference({ context: 'Identity', kind: 'glossary', element: 'User' }),
    ).toMatchObject({ ok: false, error: { code: 'unknown_kind' } })
    expect(normalizeSclElementReference({ context: 'Identity', kind: 'model' })).toMatchObject({
      ok: false,
      error: { code: 'invalid_reference' },
    })
    expect(
      normalizeSclElementReference({
        context: 'Identity',
        kind: 'model',
        element: 'User',
        index: 0,
      }),
    ).toMatchObject({ ok: false, error: { code: 'invalid_reference' } })
  })

  it('resolves every supported named kind and standard requirements', async () => {
    const index = await fixtureIndex()
    const references: SclElementReference[] = [
      { context: 'Identity', kind: 'model', element: 'User' },
      { context: 'Identity', kind: 'interface', element: 'ReadUser' },
      { context: 'Identity', kind: 'state', element: 'UserLifecycle' },
      { context: 'Identity', kind: 'authorization_resource', element: 'User' },
      { context: 'Identity', kind: 'authorization_principal', element: 'User' },
      { context: 'Identity', kind: 'authorization_policy', element: 'User' },
      { context: 'Identity', kind: 'objective', element: 'ReadLatency' },
      { context: 'Identity', kind: 'scenario', element: 'Read user' },
      { context: 'Identity', kind: 'flow', element: 'UserFlow' },
      {
        context: 'System',
        kind: 'standard_requirement',
        standard: 'ExampleStandard',
        requirement: 'REQ-1',
      },
    ]
    for (const reference of references) {
      expect(resolveSclElementReference(index, reference)).toMatchObject({
        ok: true,
        reference,
        canonical: canonicalSclElementReference(reference),
      })
    }
  })

  it('rejects unknown context, element, standard, and requirement', async () => {
    const index = await fixtureIndex()
    expect(
      resolveSclElementReference(index, { context: 'Missing', kind: 'model', element: 'User' }),
    ).toMatchObject({ ok: false, error: { code: 'unknown_context' } })
    expect(
      resolveSclElementReference(index, {
        context: 'Identity',
        kind: 'model',
        element: 'Missing',
      }),
    ).toMatchObject({ ok: false, error: { code: 'unknown_element' } })
    expect(
      resolveSclElementReference(index, {
        context: 'System',
        kind: 'standard_requirement',
        standard: 'Missing',
        requirement: 'REQ-1',
      }),
    ).toMatchObject({ ok: false, error: { code: 'unknown_standard' } })
    expect(
      resolveSclElementReference(index, {
        context: 'System',
        kind: 'standard_requirement',
        standard: 'ExampleStandard',
        requirement: 'Missing',
      }),
    ).toMatchObject({ ok: false, error: { code: 'unknown_requirement' } })
  })

  it('rejects unavailable, mismatched, extra, and duplicate context data', async () => {
    const root = await yaml('root.yaml')
    const identity = await yaml('contexts/identity.yaml')
    expect(buildSclWorkspaceIndex(root, { Identity: identity })).toMatchObject({
      ok: false,
      errors: [{ code: 'unavailable_context_spec' }],
    })
    expect(
      buildSclWorkspaceIndex(root, {
        Identity: { ...(identity as object), context: 'Wrong' },
        System: await yaml('contexts/system.yaml'),
      }),
    ).toMatchObject({ ok: false, errors: [{ code: 'context_mismatch' }] })
    expect(
      buildSclWorkspaceIndex(root, {
        Identity: identity,
        System: await yaml('contexts/system.yaml'),
        Extra: identity,
      }),
    ).toMatchObject({ ok: false, errors: [{ code: 'unknown_context' }] })

    const duplicate = {
      system: 'demo',
      spec_version: '3.0',
      context: 'System',
      standards: {
        Standard: { requirements: [{ id: 'REQ' }, { id: 'REQ' }] },
      },
    }
    expect(
      buildSclWorkspaceIndex(
        { context_map: { System: {} } },
        { System: duplicate },
      ),
    ).toMatchObject({ ok: false, errors: [{ code: 'duplicate_requirement' }] })
  })

  it('derives reversible canonical text and collision-free anchors from the tuple', () => {
    const slash: SclElementReference = {
      context: 'A/B',
      kind: 'scenario',
      element: 'Foo~Bar/Baz',
    }
    expect(canonicalSclElementReference(slash)).toBe('A~1B/scenario/Foo~0Bar~1Baz')
    expect(sclElementAnchor(slash)).toBe('scl-A~2FB/scenario/Foo~7EBar~2FBaz')
    expect(
      sclElementAnchor({ context: 'A', kind: 'model', element: 'Foo Bar' }),
    ).not.toBe(sclElementAnchor({ context: 'A', kind: 'model', element: 'foo-bar' }))
  })
})
