import { describe, expect, it } from 'bun:test'
import type { ResourceTypeDefinition } from '../../types'
import {
  diffModels,
  formatSubjectForm,
  parseModelJson,
  parseSubjectForm,
  toModelDraft,
  toResourceTypes,
  validateModelDraft,
  type ModelDraft,
} from './authorizationModel'

const referenceModel: ResourceTypeDefinition[] = [
  { name: 'user', relations: [] },
  {
    name: 'group',
    relations: [{ name: 'member', rewrites: [{ kind: 'direct', direct_subject_types: ['user'] }] }],
  },
  {
    name: 'folder',
    relations: [{ name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['user'] }] }],
  },
  {
    name: 'document',
    relations: [
      { name: 'parent', rewrites: [{ kind: 'direct', direct_subject_types: ['folder'] }] },
      {
        name: 'editor',
        rewrites: [{ kind: 'direct', direct_subject_types: ['user', 'group#member', 'user:*'] }],
      },
      {
        name: 'viewer',
        rewrites: [
          { kind: 'computed_userset', computed_relation: 'editor' },
          { kind: 'tuple_to_userset', tupleset_relation: 'parent', computed_relation: 'viewer' },
        ],
      },
    ],
  },
]

function draftOf(types: ResourceTypeDefinition[]): ModelDraft {
  const draft = toModelDraft(types)
  if (!draft) throw new Error('the fixture must be representable')
  return draft
}

function issueCodes(types: ResourceTypeDefinition[]): string[] {
  return validateModelDraft(draftOf(types)).map((issue) => issue.code)
}

function withDocumentRewrite(rewrite: ResourceTypeDefinition['relations'][number]['rewrites'][0]) {
  return [
    { name: 'user', relations: [] },
    {
      name: 'document',
      relations: [
        { name: 'owner', rewrites: [{ kind: 'direct', direct_subject_types: ['user'] }] },
        { name: 'group_like', rewrites: [{ kind: 'direct', direct_subject_types: ['user:*'] }] },
        { name: 'viewer', rewrites: [rewrite] },
      ],
    },
  ] satisfies ResourceTypeDefinition[]
}

describe('subject forms', () => {
  it('keeps the three forms when parsed and formatted', () => {
    expect(parseSubjectForm('user')).toEqual({ kind: 'object', type: 'user' })
    expect(parseSubjectForm('group#member')).toEqual({
      kind: 'subject_set',
      type: 'group',
      relation: 'member',
    })
    expect(parseSubjectForm('user:*')).toEqual({ kind: 'wildcard', type: 'user' })
    for (const form of ['user', 'group#member', 'user:*']) {
      expect(formatSubjectForm(parseSubjectForm(form))).toBe(form)
    }
  })
})

describe('model conversion', () => {
  it('returns an equal model after a round trip through the draft', () => {
    expect(toResourceTypes(draftOf(referenceModel))).toEqual(referenceModel)
  })

  it('refuses what the form cannot represent instead of dropping it', () => {
    const cases: unknown[] = [
      {},
      [{ name: 'a', relations: [], extra: true }],
      [{ name: 'a', relations: [{ name: 'r', rewrites: [{ kind: 'intersection' }] }] }],
      [
        {
          name: 'a',
          relations: [
            {
              name: 'r',
              rewrites: [
                { kind: 'direct', direct_subject_types: ['user'], computed_relation: 'x' },
              ],
            },
          ],
        },
      ],
      [{ name: 'a', relations: [{ name: 'r', rewrites: [{ kind: 'computed_userset' }] }] }],
    ]
    for (const input of cases) {
      expect(toModelDraft(input)).toBeNull()
    }
  })

  it('reads only JSON that is a representable model', () => {
    expect(parseModelJson('not json')).toBeNull()
    expect(parseModelJson('{"resource_types":[]}')).toBeNull()
    expect(parseModelJson(JSON.stringify(referenceModel))).toEqual(draftOf(referenceModel))
  })
})

describe('validateModelDraft', () => {
  it('accepts a coherent model', () => {
    expect(issueCodes(referenceModel)).toEqual([])
  })

  it('requires at least one resource type', () => {
    expect(issueCodes([])).toEqual(['no_resource_types'])
  })

  it('accepts a 64 character name and refuses 65 characters', () => {
    expect(issueCodes([{ name: 'a'.repeat(64), relations: [] }])).toEqual([])
    expect(issueCodes([{ name: 'a'.repeat(65), relations: [] }])).toEqual(['invalid_name'])
  })

  it('refuses names outside the format', () => {
    expect(issueCodes([{ name: 'Document', relations: [] }])).toEqual(['invalid_name'])
    expect(issueCodes([{ name: '1doc', relations: [] }])).toEqual(['invalid_name'])
  })

  it('refuses duplicated types and relations', () => {
    expect(
      issueCodes([
        { name: 'a', relations: [] },
        { name: 'a', relations: [] },
      ]),
    ).toEqual(['duplicate_type'])
    const direct = { kind: 'direct' as const, direct_subject_types: ['a'] }
    expect(
      issueCodes([
        {
          name: 'a',
          relations: [
            { name: 'r', rewrites: [direct] },
            { name: 'r', rewrites: [direct] },
          ],
        },
      ]),
    ).toEqual(['duplicate_relation'])
  })

  it('refuses a relation without rewrites and a direct rewrite without subjects', () => {
    expect(issueCodes([{ name: 'a', relations: [{ name: 'r', rewrites: [] }] }])).toEqual([
      'no_rewrites',
    ])
    expect(
      issueCodes([{ name: 'a', relations: [{ name: 'r', rewrites: [{ kind: 'direct' }] }] }]),
    ).toEqual(['no_subject_forms'])
  })

  it('refuses a computed userset that references itself or an undeclared relation', () => {
    expect(
      issueCodes(withDocumentRewrite({ kind: 'computed_userset', computed_relation: 'viewer' })),
    ).toEqual(['self_reference'])
    expect(
      issueCodes(withDocumentRewrite({ kind: 'computed_userset', computed_relation: 'missing' })),
    ).toEqual(['unknown_relation'])
  })

  it('leaves a longer computed cycle to the server', () => {
    expect(
      issueCodes([
        {
          name: 'a',
          relations: [
            { name: 'r', rewrites: [{ kind: 'computed_userset', computed_relation: 's' }] },
            { name: 's', rewrites: [{ kind: 'computed_userset', computed_relation: 'r' }] },
          ],
        },
      ]),
    ).toEqual([])
  })

  it('requires a tupleset that accepts plain objects declaring the computed relation', () => {
    expect(
      issueCodes(
        withDocumentRewrite({
          kind: 'tuple_to_userset',
          tupleset_relation: 'missing',
          computed_relation: 'owner',
        }),
      ),
    ).toEqual(['unknown_relation'])
    expect(
      issueCodes(
        withDocumentRewrite({
          kind: 'tuple_to_userset',
          tupleset_relation: 'group_like',
          computed_relation: 'owner',
        }),
      ),
    ).toEqual(['tupleset_without_object'])
    expect(
      issueCodes(
        withDocumentRewrite({
          kind: 'tuple_to_userset',
          tupleset_relation: 'owner',
          computed_relation: 'owner',
        }),
      ),
    ).toEqual(['unknown_relation'])
  })

  it('refuses subject forms that name undeclared types or relations', () => {
    for (const form of ['ghost', 'ghost:*', 'ghost#member', 'user#ghost']) {
      expect(
        issueCodes([
          { name: 'user', relations: [] },
          {
            name: 'a',
            relations: [
              { name: 'r', rewrites: [{ kind: 'direct', direct_subject_types: [form] }] },
            ],
          },
        ]),
      ).toEqual(['unknown_subject'])
    }
  })
})

describe('diffModels', () => {
  const before: ResourceTypeDefinition[] = [
    { name: 'user', relations: [] },
    {
      name: 'doc',
      relations: [
        {
          name: 'viewer',
          rewrites: [{ kind: 'direct', direct_subject_types: ['user', 'user:*'] }],
        },
      ],
    },
  ]

  it('reports nothing when only the order differs', () => {
    const reordered: ResourceTypeDefinition[] = [
      {
        name: 'doc',
        relations: [
          {
            name: 'viewer',
            rewrites: [{ kind: 'direct', direct_subject_types: ['user:*', 'user'] }],
          },
        ],
      },
      { name: 'user', relations: [] },
    ]
    expect(diffModels(before, reordered)).toEqual([])
  })

  it('reports a removed subject form and an added rule on the same relation', () => {
    const after: ResourceTypeDefinition[] = [
      { name: 'user', relations: [] },
      {
        name: 'doc',
        relations: [
          { name: 'editor', rewrites: [{ kind: 'direct', direct_subject_types: ['user'] }] },
          {
            name: 'viewer',
            rewrites: [
              { kind: 'direct', direct_subject_types: ['user'] },
              { kind: 'computed_userset', computed_relation: 'editor' },
            ],
          },
        ],
      },
    ]
    expect(diffModels(before, after)).toEqual([
      { change: 'added', target: 'relation', type: 'doc', relation: 'editor' },
      { change: 'added', target: 'rule', type: 'doc', relation: 'editor', rule: 'direct user' },
      {
        change: 'added',
        target: 'rule',
        type: 'doc',
        relation: 'viewer',
        rule: 'computed_userset editor',
      },
      { change: 'removed', target: 'rule', type: 'doc', relation: 'viewer', rule: 'direct user:*' },
    ])
  })

  it('reports everything under a removed type and treats a first publication as additions', () => {
    expect(diffModels(before, [{ name: 'user', relations: [] }])).toEqual([
      { change: 'removed', target: 'type', type: 'doc' },
      { change: 'removed', target: 'relation', type: 'doc', relation: 'viewer' },
      { change: 'removed', target: 'rule', type: 'doc', relation: 'viewer', rule: 'direct user' },
      { change: 'removed', target: 'rule', type: 'doc', relation: 'viewer', rule: 'direct user:*' },
    ])
    expect(diffModels([], [{ name: 'user', relations: [] }])).toEqual([
      { change: 'added', target: 'type', type: 'user' },
    ])
  })
})
