import {
  getDoc,
  getFormat,
  getTags,
  getMaxItemsAsNumeric,
  getMaxLengthAsNumeric,
  getMaxValueAsNumeric,
  getMinItemsAsNumeric,
  getMinLengthAsNumeric,
  getMinValueAsNumeric,
  getNamespaceFullName,
  getPattern,
  getSourceLocation,
  getTypeName,
  type Enum,
  type Model,
  type ModelProperty,
  type Namespace,
  type Program,
  type Scalar,
  type Type,
  type Union,
  type Value,
} from '@typespec/compiler'
import { SyntaxKind } from '@typespec/compiler/ast'
import { relative } from 'node:path'

export type CatalogProperty = {
  name: string
  type: string
  optional: boolean
  doc?: string
  default?: string
  constraints: string[]
  references: string[]
}

export type CatalogMember = {
  name: string
  value?: string
  type?: string
  doc?: string
}

export type CatalogSymbol = {
  kind: 'model' | 'enum' | 'union' | 'scalar'
  name: string
  namespace: string
  shortName: string
  doc?: string
  apiExposed: boolean
  base?: string
  properties: CatalogProperty[]
  members: CatalogMember[]
  references: string[]
  /** The bounded context whose directory declares the symbol, when it has one. */
  context?: string
}

export type TypeSpecCatalog = {
  symbols: CatalogSymbol[]
  /** OpenAPI tag names, by the bounded context that declares them. */
  contextTags: Record<string, string[]>
}

/**
 * The owning context of a declaration is where its source file sits. The
 * standard layout already says it, so no table has to repeat it.
 */
function declaringContext(
  type: Model | Enum | Union | Scalar | Namespace,
  repositoryRoot: string,
): string | undefined {
  const node = type.node
  if (!node) return undefined
  const path = relative(repositoryRoot, getSourceLocation(node).file.path).replaceAll('\\', '/')
  return path.match(/^spec\/contexts\/([^/]+)\//)?.[1]
}

function namespaceName(namespace: Namespace | undefined): string {
  return namespace ? getNamespaceFullName(namespace) : ''
}

function fullName(type: Model | Enum | Union | Scalar): string {
  const namespace = namespaceName(type.namespace)
  const name = type.name ?? ''
  return namespace ? `${namespace}.${name}` : name
}

function displayValue(value: Value | undefined): string | undefined {
  if (!value) return undefined
  switch (value.valueKind) {
    case 'StringValue':
      return JSON.stringify(value.value)
    case 'NumericValue':
    case 'BooleanValue':
      return String(value.value)
    case 'NullValue':
      return 'null'
    case 'EnumValue':
      return `${fullName(value.value.enum)}.${value.value.name}`
    case 'ArrayValue':
      return `[${value.values.map((item) => displayValue(item) ?? '?').join(', ')}]`
    case 'ObjectValue':
      return `{ ${[...value.properties.values()]
        .map((property) => `${property.name}: ${displayValue(property.value) ?? '?'}`)
        .join(', ')} }`
    case 'ScalarValue':
      return `${fullName(value.scalar)}(${value.value.args
        .map((argument) => displayValue(argument) ?? '?')
        .join(', ')})`
    case 'Function':
      return value.name ?? '(function)'
  }
}

function constraints(program: Program, type: Type): string[] {
  const entries: Array<[string, { toString(): string } | string | number | undefined]> = [
    ['format', getFormat(program, type)],
    ['pattern', getPattern(program, type)],
    ['minLength', getMinLengthAsNumeric(program, type)],
    ['maxLength', getMaxLengthAsNumeric(program, type)],
    ['minItems', getMinItemsAsNumeric(program, type)],
    ['maxItems', getMaxItemsAsNumeric(program, type)],
    ['minValue', getMinValueAsNumeric(program, type)],
    ['maxValue', getMaxValueAsNumeric(program, type)],
  ]
  return entries
    .filter(
      (entry): entry is [string, { toString(): string } | string | number] =>
        entry[1] !== undefined,
    )
    .map(([name, value]) => `${name}: ${value}`)
}

function namedReference(type: Type, projectTypes: Set<Type>): string | undefined {
  if (
    (type.kind === 'Model' ||
      type.kind === 'Enum' ||
      type.kind === 'Union' ||
      type.kind === 'Scalar') &&
    type.name &&
    projectTypes.has(type)
  ) {
    return fullName(type)
  }
  return undefined
}

function collectReferences(
  type: Type,
  references: Set<string>,
  projectTypes: Set<Type>,
  seen = new Set<Type>(),
): void {
  if (seen.has(type)) return
  seen.add(type)
  const direct = namedReference(type, projectTypes)
  if (direct) {
    references.add(direct)
    return
  }
  switch (type.kind) {
    case 'Model':
      if (type.indexer) collectReferences(type.indexer.value, references, projectTypes, seen)
      for (const property of type.properties.values())
        collectReferences(property.type, references, projectTypes, seen)
      break
    case 'ModelProperty':
      collectReferences(type.type, references, projectTypes, seen)
      break
    case 'Union':
      for (const variant of type.variants.values())
        collectReferences(variant.type, references, projectTypes, seen)
      break
    case 'UnionVariant':
      collectReferences(type.type, references, projectTypes, seen)
      break
    case 'Tuple':
      for (const value of type.values) collectReferences(value, references, projectTypes, seen)
      break
  }
}

function property(
  program: Program,
  value: ModelProperty,
  projectTypes: Set<Type>,
): CatalogProperty {
  const references = new Set<string>()
  collectReferences(value.type, references, projectTypes)
  return {
    name: value.name,
    type: getTypeName(value.type),
    optional: value.optional,
    doc: getDoc(program, value),
    default: displayValue(value.defaultValue),
    constraints: constraints(program, value),
    references: [...references].sort(),
  }
}

function isApiExposed(name: string, schemas: Set<string>): boolean {
  return [...schemas].some((schema) => name === schema || name.endsWith(`.${schema}`))
}

function modelSymbol(
  program: Program,
  value: Model,
  schemas: Set<string>,
  projectTypes: Set<Type>,
): CatalogSymbol {
  const name = fullName(value)
  const references = new Set<string>()
  if (value.baseModel && projectTypes.has(value.baseModel))
    references.add(fullName(value.baseModel))
  if (value.sourceModel && projectTypes.has(value.sourceModel))
    references.add(fullName(value.sourceModel))
  const properties = [...value.properties.values()].map((item) =>
    property(program, item, projectTypes),
  )
  for (const item of properties) for (const reference of item.references) references.add(reference)
  references.delete(name)
  return {
    kind: 'model',
    name,
    namespace: namespaceName(value.namespace),
    shortName: value.name,
    doc: getDoc(program, value),
    apiExposed: isApiExposed(name, schemas),
    base:
      value.baseModel && projectTypes.has(value.baseModel) ? fullName(value.baseModel) : undefined,
    properties,
    members: [],
    references: [...references].sort(),
  }
}

function enumSymbol(program: Program, value: Enum, schemas: Set<string>): CatalogSymbol {
  const name = fullName(value)
  return {
    kind: 'enum',
    name,
    namespace: namespaceName(value.namespace),
    shortName: value.name,
    doc: getDoc(program, value),
    apiExposed: isApiExposed(name, schemas),
    properties: [],
    members: [...value.members.values()].map((member) => ({
      name: member.name,
      value: member.value === undefined ? undefined : String(member.value),
      doc: getDoc(program, member),
    })),
    references: [],
  }
}

function unionSymbol(
  program: Program,
  value: Union,
  schemas: Set<string>,
  projectTypes: Set<Type>,
): CatalogSymbol {
  const name = fullName(value)
  const references = new Set<string>()
  const members = [...value.variants.entries()].map(([key, variant]) => {
    collectReferences(variant.type, references, projectTypes)
    return {
      name:
        typeof key === 'symbol'
          ? typeof variant.name === 'symbol'
            ? '(variant)'
            : (variant.name ?? '(variant)')
          : String(key),
      type: getTypeName(variant.type),
      doc: getDoc(program, variant),
    }
  })
  references.delete(name)
  return {
    kind: 'union',
    name,
    namespace: namespaceName(value.namespace),
    shortName: value.name ?? '(anonymous)',
    doc: getDoc(program, value),
    apiExposed: isApiExposed(name, schemas),
    properties: [],
    members,
    references: [...references].sort(),
  }
}

function scalarSymbol(
  program: Program,
  value: Scalar,
  schemas: Set<string>,
  projectTypes: Set<Type>,
): CatalogSymbol {
  const name = fullName(value)
  const base = value.baseScalar ? getTypeName(value.baseScalar) : undefined
  return {
    kind: 'scalar',
    name,
    namespace: namespaceName(value.namespace),
    shortName: value.name,
    doc: getDoc(program, value),
    apiExposed: isApiExposed(name, schemas),
    base,
    properties: [],
    members: [],
    references: value.baseScalar && projectTypes.has(value.baseScalar) && base ? [base] : [],
  }
}

function isProjectDeclaration(program: Program, value: Type): boolean {
  if (
    value.kind !== 'Model' &&
    value.kind !== 'Enum' &&
    value.kind !== 'Union' &&
    value.kind !== 'Scalar'
  )
    return false
  if (!value.node || ('templateMapper' in value && value.templateMapper)) return false
  if (!('id' in value.node) || !value.node.id || value.name !== value.node.id.sv) return false
  const declarationKind =
    (value.kind === 'Model' && value.node.kind === SyntaxKind.ModelStatement) ||
    (value.kind === 'Enum' && value.node.kind === SyntaxKind.EnumStatement) ||
    (value.kind === 'Union' && value.node.kind === SyntaxKind.UnionStatement) ||
    (value.kind === 'Scalar' && value.node.kind === SyntaxKind.ScalarStatement)
  if (!declarationKind) return false
  return program.getSourceFileLocationContext(getSourceLocation(value.node).file).type === 'project'
}

function collectProjectTypes(program: Program, namespace: Namespace, output: Set<Type>): void {
  if (namespace.name !== 'Operations') {
    for (const value of namespace.models.values())
      if (value.name && isProjectDeclaration(program, value)) output.add(value)
    for (const value of namespace.enums.values())
      if (isProjectDeclaration(program, value)) output.add(value)
    for (const value of namespace.unions.values())
      if (value.name && isProjectDeclaration(program, value)) output.add(value)
    for (const value of namespace.scalars.values())
      if (isProjectDeclaration(program, value)) output.add(value)
  }
  for (const child of namespace.namespaces.values()) collectProjectTypes(program, child, output)
}

/**
 * An operation namespace carries the OpenAPI tag its context owns, so the tag a
 * reader sees in the API Reference resolves back to a context directory.
 */
function collectContextTags(
  program: Program,
  namespace: Namespace,
  repositoryRoot: string,
  output: Map<string, Set<string>>,
): void {
  const context = declaringContext(namespace, repositoryRoot)
  if (context) {
    for (const tag of getTags(program, namespace)) {
      const tags = output.get(context) ?? new Set<string>()
      tags.add(tag)
      output.set(context, tags)
    }
  }
  for (const child of namespace.namespaces.values())
    collectContextTags(program, child, repositoryRoot, output)
}

export function extractTypeSpecCatalog(
  program: Program,
  apiSchemas: Set<string>,
  repositoryRoot: string,
): TypeSpecCatalog {
  const projectTypes = new Set<Type>()
  collectProjectTypes(program, program.getGlobalNamespaceType(), projectTypes)
  const symbols: CatalogSymbol[] = []
  for (const value of projectTypes) {
    switch (value.kind) {
      case 'Model':
        symbols.push(modelSymbol(program, value, apiSchemas, projectTypes))
        break
      case 'Enum':
        symbols.push(enumSymbol(program, value, apiSchemas))
        break
      case 'Union':
        symbols.push(unionSymbol(program, value, apiSchemas, projectTypes))
        break
      case 'Scalar':
        symbols.push(scalarSymbol(program, value, apiSchemas, projectTypes))
        break
      default:
        continue
    }
    const symbol = symbols[symbols.length - 1]
    if (symbol) symbol.context = declaringContext(value, repositoryRoot)
  }
  const tags = new Map<string, Set<string>>()
  collectContextTags(program, program.getGlobalNamespaceType(), repositoryRoot, tags)
  return {
    symbols: symbols.sort((a, b) => a.name.localeCompare(b.name)),
    contextTags: Object.fromEntries(
      [...tags.entries()]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, value]) => [key, [...value].sort()]),
    ),
  }
}
