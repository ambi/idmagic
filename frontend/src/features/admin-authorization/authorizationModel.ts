import type { RelationRewrite, ResourceTypeDefinition } from '../../types'

export type SubjectFormDraft =
  | { kind: 'object'; type: string }
  | { kind: 'subject_set'; type: string; relation: string }
  | { kind: 'wildcard'; type: string }

export type RewriteDraft =
  | { kind: 'direct'; subjects: SubjectFormDraft[] }
  | { kind: 'computed_userset'; computedRelation: string }
  | { kind: 'tuple_to_userset'; tuplesetRelation: string; computedRelation: string }

export type RelationDraft = { name: string; rewrites: RewriteDraft[] }
export type ResourceTypeDraft = { name: string; relations: RelationDraft[] }
export type ModelDraft = { resourceTypes: ResourceTypeDraft[] }

export type ModelDraftIssueCode =
  | 'no_resource_types'
  | 'invalid_name'
  | 'duplicate_type'
  | 'duplicate_relation'
  | 'no_rewrites'
  | 'no_subject_forms'
  | 'self_reference'
  | 'unknown_relation'
  | 'unknown_subject'
  | 'tupleset_without_object'

export type ModelDraftIssue = { code: ModelDraftIssueCode; subject: string }

export type ModelDiffEntry = {
  change: 'added' | 'removed'
  target: 'type' | 'relation' | 'rule'
  type: string
  relation?: string
  rule?: string
}

const NAME_PATTERN = /^[a-z][a-z0-9_]*$/
const NAME_MAX_LENGTH = 64

export function isValidName(name: string): boolean {
  return NAME_PATTERN.test(name) && name.length <= NAME_MAX_LENGTH
}

export function parseSubjectForm(form: string): SubjectFormDraft {
  if (form.endsWith(':*')) return { kind: 'wildcard', type: form.slice(0, -2) }
  const separator = form.indexOf('#')
  if (separator >= 0) {
    return {
      kind: 'subject_set',
      type: form.slice(0, separator),
      relation: form.slice(separator + 1),
    }
  }
  return { kind: 'object', type: form }
}

export function formatSubjectForm(form: SubjectFormDraft): string {
  switch (form.kind) {
    case 'object':
      return form.type
    case 'subject_set':
      return `${form.type}#${form.relation}`
    case 'wildcard':
      return `${form.type}:*`
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function hasOnlyKeys(value: Record<string, unknown>, keys: string[]): boolean {
  return Object.keys(value).every((key) => keys.includes(key))
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string')
}

function toRewriteDraft(value: unknown): RewriteDraft | null {
  if (!isRecord(value)) return null
  switch (value.kind) {
    case 'direct': {
      if (!hasOnlyKeys(value, ['kind', 'direct_subject_types'])) return null
      const forms = value.direct_subject_types ?? []
      return isStringArray(forms) ? { kind: 'direct', subjects: forms.map(parseSubjectForm) } : null
    }
    case 'computed_userset':
      if (!hasOnlyKeys(value, ['kind', 'computed_relation'])) return null
      return typeof value.computed_relation === 'string'
        ? { kind: 'computed_userset', computedRelation: value.computed_relation }
        : null
    case 'tuple_to_userset':
      if (!hasOnlyKeys(value, ['kind', 'tupleset_relation', 'computed_relation'])) return null
      return typeof value.tupleset_relation === 'string' &&
        typeof value.computed_relation === 'string'
        ? {
            kind: 'tuple_to_userset',
            tuplesetRelation: value.tupleset_relation,
            computedRelation: value.computed_relation,
          }
        : null
    default:
      return null
  }
}

function mapAll<T>(values: unknown, convert: (value: unknown) => T | null): T[] | null {
  if (!Array.isArray(values)) return null
  const converted: T[] = []
  for (const value of values) {
    const item = convert(value)
    if (item === null) return null
    converted.push(item)
  }
  return converted
}

function toRelationDraft(value: unknown): RelationDraft | null {
  if (!isRecord(value) || !hasOnlyKeys(value, ['name', 'rewrites'])) return null
  const rewrites = mapAll(value.rewrites, toRewriteDraft)
  return typeof value.name === 'string' && rewrites ? { name: value.name, rewrites } : null
}

function toResourceTypeDraft(value: unknown): ResourceTypeDraft | null {
  if (!isRecord(value) || !hasOnlyKeys(value, ['name', 'relations'])) return null
  const relations = mapAll(value.relations, toRelationDraft)
  return typeof value.name === 'string' && relations ? { name: value.name, relations } : null
}

// toModelDraft はフォームで表せない入力に null を返す。黙って捨てると、JSON とフォームを
// 往復しただけで発行する内容が変わるためである。
export function toModelDraft(resourceTypes: unknown): ModelDraft | null {
  const converted = mapAll(resourceTypes, toResourceTypeDraft)
  return converted && { resourceTypes: converted }
}

function toRewrite(draft: RewriteDraft): RelationRewrite {
  switch (draft.kind) {
    case 'direct':
      return { kind: 'direct', direct_subject_types: draft.subjects.map(formatSubjectForm) }
    case 'computed_userset':
      return { kind: 'computed_userset', computed_relation: draft.computedRelation }
    case 'tuple_to_userset':
      return {
        kind: 'tuple_to_userset',
        tupleset_relation: draft.tuplesetRelation,
        computed_relation: draft.computedRelation,
      }
  }
}

export function toResourceTypes(draft: ModelDraft): ResourceTypeDefinition[] {
  return draft.resourceTypes.map((type) => ({
    name: type.name,
    relations: type.relations.map((relation) => ({
      name: relation.name,
      rewrites: relation.rewrites.map(toRewrite),
    })),
  }))
}

export function parseModelJson(raw: string): ModelDraft | null {
  try {
    return toModelDraft(JSON.parse(raw))
  } catch {
    return null
  }
}

function findRelation(
  draft: ModelDraft,
  type: string,
  relation: string,
): RelationDraft | undefined {
  return draft.resourceTypes
    .find((candidate) => candidate.name === type)
    ?.relations.find((candidate) => candidate.name === relation)
}

function duplicates(names: string[]): string[] {
  return [...new Set(names.filter((name, index) => names.indexOf(name) !== index))]
}

function isDeclaredSubject(draft: ModelDraft, form: SubjectFormDraft): boolean {
  if (form.kind === 'subject_set')
    return findRelation(draft, form.type, form.relation) !== undefined
  return draft.resourceTypes.some((type) => type.name === form.type)
}

function validateRewrite(
  draft: ModelDraft,
  type: ResourceTypeDraft,
  relation: RelationDraft,
  rewrite: RewriteDraft,
): ModelDraftIssue[] {
  const subject = `${type.name}#${relation.name}`
  switch (rewrite.kind) {
    case 'direct':
      if (rewrite.subjects.length === 0) return [{ code: 'no_subject_forms', subject }]
      return rewrite.subjects
        .filter((form) => !isDeclaredSubject(draft, form))
        .map((form) => ({ code: 'unknown_subject', subject: formatSubjectForm(form) }))
    case 'computed_userset':
      if (rewrite.computedRelation === relation.name) return [{ code: 'self_reference', subject }]
      return findRelation(draft, type.name, rewrite.computedRelation)
        ? []
        : [{ code: 'unknown_relation', subject: `${type.name}#${rewrite.computedRelation}` }]
    case 'tuple_to_userset': {
      const tupleset = findRelation(draft, type.name, rewrite.tuplesetRelation)
      if (!tupleset) {
        return [{ code: 'unknown_relation', subject: `${type.name}#${rewrite.tuplesetRelation}` }]
      }
      const targets = tupleset.rewrites.flatMap((candidate) =>
        candidate.kind === 'direct'
          ? candidate.subjects.filter((form) => form.kind === 'object').map((form) => form.type)
          : [],
      )
      if (targets.length === 0) {
        return [{ code: 'tupleset_without_object', subject: `${type.name}#${tupleset.name}` }]
      }
      return targets.some((target) => findRelation(draft, target, rewrite.computedRelation))
        ? []
        : [
            {
              code: 'unknown_relation',
              subject: `${targets.join(', ')}#${rewrite.computedRelation}`,
            },
          ]
    }
  }
}

// validateModelDraft はフォームで防げる不備だけを拾う。computed_userset の連なりの循環は
// サーバーの検証に任せて 422 を見せ、fail-closed の判断を二か所に持たない。
export function validateModelDraft(draft: ModelDraft): ModelDraftIssue[] {
  if (draft.resourceTypes.length === 0) return [{ code: 'no_resource_types', subject: '' }]
  const issues: ModelDraftIssue[] = duplicates(draft.resourceTypes.map((type) => type.name)).map(
    (name) => ({ code: 'duplicate_type', subject: name }),
  )
  for (const type of draft.resourceTypes) {
    if (!isValidName(type.name)) issues.push({ code: 'invalid_name', subject: type.name })
    for (const name of duplicates(type.relations.map((relation) => relation.name))) {
      issues.push({ code: 'duplicate_relation', subject: `${type.name}#${name}` })
    }
    for (const relation of type.relations) {
      if (!isValidName(relation.name)) {
        issues.push({ code: 'invalid_name', subject: `${type.name}#${relation.name}` })
      }
      if (relation.rewrites.length === 0) {
        issues.push({ code: 'no_rewrites', subject: `${type.name}#${relation.name}` })
      }
      for (const rewrite of relation.rewrites) {
        issues.push(...validateRewrite(draft, type, relation, rewrite))
      }
    }
  }
  return issues
}

function ruleLabels(rewrite: RelationRewrite): string[] {
  switch (rewrite.kind) {
    case 'direct':
      return (rewrite.direct_subject_types ?? []).map((form) => `direct ${form}`)
    case 'computed_userset':
      return [`computed_userset ${rewrite.computed_relation}`]
    case 'tuple_to_userset':
      return [`tuple_to_userset ${rewrite.tupleset_relation} -> ${rewrite.computed_relation}`]
  }
}

function typeEntries(
  type: ResourceTypeDefinition,
  other: ResourceTypeDefinition | undefined,
  change: ModelDiffEntry['change'],
): ModelDiffEntry[] {
  const entries: ModelDiffEntry[] = other ? [] : [{ change, target: 'type', type: type.name }]
  for (const relation of type.relations) {
    const otherRelation = other?.relations.find((candidate) => candidate.name === relation.name)
    if (!otherRelation) {
      entries.push({ change, target: 'relation', type: type.name, relation: relation.name })
    }
    const otherRules = new Set(otherRelation?.rewrites.flatMap(ruleLabels))
    for (const rule of relation.rewrites.flatMap(ruleLabels)) {
      if (!otherRules.has(rule)) {
        entries.push({ change, target: 'rule', type: type.name, relation: relation.name, rule })
      }
    }
  }
  return entries
}

function sideEntries(
  source: ResourceTypeDefinition[],
  other: ResourceTypeDefinition[],
  change: ModelDiffEntry['change'],
): ModelDiffEntry[] {
  return source.flatMap((type) =>
    typeEntries(
      type,
      other.find((candidate) => candidate.name === type.name),
      change,
    ),
  )
}

// diffModels は規則を順序によらない集合として比べる。direct は主体の形ごとに 1 規則とする。
export function diffModels(
  before: ResourceTypeDefinition[],
  after: ResourceTypeDefinition[],
): ModelDiffEntry[] {
  return [...sideEntries(after, before, 'added'), ...sideEntries(before, after, 'removed')]
}
