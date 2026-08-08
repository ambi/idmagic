/**
 * Pure transform: two OpenAPI 3.1 documents (a frozen release baseline and
 * the freshly generated current spec) -> a list of breaking-change findings.
 *
 * Implements the compatibility definition from ADR-156: additive changes
 * (new fields, new optional parameters, new endpoints, new error codes) are
 * fine; removing or renaming a field, changing a field's type, making a
 * field required, changing a default, or removing a documented error code
 * are all breaking.
 */

export type JsonSchema = Record<string, unknown>
type PathItem = Record<string, JsonSchema>
type OpenApiDoc = {
  paths?: Record<string, PathItem>
  components?: { schemas?: Record<string, JsonSchema> }
}

export interface CompatFinding {
  operation: string
  message: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function refName(schema: unknown, prefix = '#/components/schemas/'): string | undefined {
  if (!isRecord(schema)) return undefined
  const ref = schema.$ref
  return typeof ref === 'string' && ref.startsWith(prefix) ? ref.slice(prefix.length) : undefined
}

/** Resolve a `$ref` against components, cycle-guarded. Non-refs pass through unchanged. */
function resolve(
  schema: JsonSchema | undefined,
  components: Record<string, JsonSchema>,
  seen: ReadonlySet<string>,
): JsonSchema {
  if (!schema) return {}
  const name = refName(schema)
  if (name === undefined) return schema
  if (seen.has(name)) return {} // self/mutual recursion guard: stop expanding
  const target = components[name]
  return target ? resolve(target, components, new Set(seen).add(name)) : {}
}

function diffSchema(
  ctx: string,
  base: JsonSchema | undefined,
  curr: JsonSchema | undefined,
  baseComponents: Record<string, JsonSchema>,
  currComponents: Record<string, JsonSchema>,
  findings: CompatFinding[],
  baseSeen: ReadonlySet<string> = new Set(),
  currSeen: ReadonlySet<string> = new Set(),
): void {
  const baseRef = base ? refName(base) : undefined
  const currRef = curr ? refName(curr) : undefined
  if (baseRef !== undefined && baseSeen.has(baseRef)) return // cycle: already compared this branch
  const b = resolve(base, baseComponents, baseSeen)
  const c = resolve(curr, currComponents, currSeen)
  if (Object.keys(b).length === 0) return
  const nextBaseSeen = baseRef !== undefined ? new Set(baseSeen).add(baseRef) : baseSeen
  const nextCurrSeen = currRef !== undefined ? new Set(currSeen).add(currRef) : currSeen

  if (typeof b.type === 'string' && typeof c.type === 'string' && b.type !== c.type) {
    findings.push({ operation: ctx, message: `type changed from '${b.type}' to '${c.type}'` })
  }
  if ('default' in b && JSON.stringify(b.default) !== JSON.stringify(c.default)) {
    findings.push({
      operation: ctx,
      message: `default value changed from ${JSON.stringify(b.default)} to ${JSON.stringify(c.default)}`,
    })
  }

  const baseRequired = new Set(Array.isArray(b.required) ? (b.required as string[]) : [])
  const currRequired = new Set(Array.isArray(c.required) ? (c.required as string[]) : [])
  for (const name of currRequired) {
    if (!baseRequired.has(name)) {
      findings.push({ operation: ctx, message: `field '${name}' became required` })
    }
  }

  const baseProps = isRecord(b.properties) ? (b.properties as Record<string, JsonSchema>) : {}
  const currProps = isRecord(c.properties) ? (c.properties as Record<string, JsonSchema>) : {}
  for (const [name, baseField] of Object.entries(baseProps)) {
    if (!(name in currProps)) {
      findings.push({ operation: ctx, message: `field '${name}' removed` })
      continue
    }
    diffSchema(
      `${ctx}.${name}`,
      baseField,
      currProps[name],
      baseComponents,
      currComponents,
      findings,
      nextBaseSeen,
      nextCurrSeen,
    )
  }

  if (Array.isArray(b.enum)) {
    const currEnum = new Set(Array.isArray(c.enum) ? (c.enum as unknown[]) : [])
    for (const value of b.enum as unknown[]) {
      if (!currEnum.has(value)) {
        findings.push({ operation: ctx, message: `enum value ${JSON.stringify(value)} removed` })
      }
    }
  }

  if (Array.isArray(b.oneOf)) {
    const baseNames = (b.oneOf as unknown[]).map((s) => refName(s)).filter((n): n is string => !!n)
    const currNames = new Set(
      (Array.isArray(c.oneOf) ? (c.oneOf as unknown[]) : [])
        .map((s) => refName(s))
        .filter((n): n is string => !!n),
    )
    for (const name of baseNames) {
      if (!currNames.has(name)) {
        findings.push({ operation: ctx, message: `error code '${name}' removed` })
      }
    }
  }

  if (b.items) {
    diffSchema(
      `${ctx}[]`,
      b.items as JsonSchema,
      c.items as JsonSchema,
      baseComponents,
      currComponents,
      findings,
      nextBaseSeen,
      nextCurrSeen,
    )
  }
}

function firstContentSchema(container: unknown): JsonSchema | undefined {
  if (!isRecord(container)) return undefined
  const content = container.content
  if (!isRecord(content)) return undefined
  const first = Object.values(content)[0]
  return isRecord(first) && isRecord(first.schema) ? (first.schema as JsonSchema) : undefined
}

function diffParameters(
  ctx: string,
  baseOp: JsonSchema,
  currOp: JsonSchema,
  baseComponents: Record<string, JsonSchema>,
  currComponents: Record<string, JsonSchema>,
  findings: CompatFinding[],
): void {
  const baseParams = Array.isArray(baseOp.parameters) ? (baseOp.parameters as JsonSchema[]) : []
  const currByKey = new Map(
    (Array.isArray(currOp.parameters) ? (currOp.parameters as JsonSchema[]) : []).map((p) => [
      `${p.in}:${p.name}`,
      p,
    ]),
  )
  for (const baseParam of baseParams) {
    const key = `${baseParam.in}:${baseParam.name}`
    const currParam = currByKey.get(key)
    if (!currParam) {
      findings.push({ operation: ctx, message: `parameter '${baseParam.name}' removed` })
      continue
    }
    if (!baseParam.required && currParam.required) {
      findings.push({ operation: ctx, message: `parameter '${baseParam.name}' became required` })
    }
    diffSchema(
      `${ctx} parameter ${String(baseParam.name)}`,
      baseParam.schema as JsonSchema,
      currParam.schema as JsonSchema,
      baseComponents,
      currComponents,
      findings,
    )
  }
}

/** Compare a baseline OpenAPI document against the current one; breaking changes only (ADR-156). */
export function compareOpenApi(baseline: JsonSchema, current: JsonSchema): CompatFinding[] {
  const findings: CompatFinding[] = []
  const base = baseline as OpenApiDoc
  const curr = current as OpenApiDoc
  const baseComponents = base.components?.schemas ?? {}
  const currComponents = curr.components?.schemas ?? {}
  const basePaths = base.paths ?? {}
  const currPaths = curr.paths ?? {}

  for (const [path, baseMethods] of Object.entries(basePaths)) {
    const currMethods = currPaths[path]
    if (!currMethods) {
      findings.push({ operation: `* ${path}`, message: 'path removed' })
      continue
    }
    for (const [method, baseOp] of Object.entries(baseMethods)) {
      const ctx = `${method.toUpperCase()} ${path}`
      const currOp = currMethods[method]
      if (!currOp) {
        findings.push({ operation: ctx, message: 'operation removed' })
        continue
      }

      diffParameters(ctx, baseOp, currOp, baseComponents, currComponents, findings)

      const baseRequestSchema = firstContentSchema(baseOp.requestBody)
      if (baseRequestSchema) {
        const currRequestSchema = firstContentSchema(currOp.requestBody)
        if (!currRequestSchema) {
          findings.push({ operation: `${ctx} request`, message: 'request body removed' })
        } else {
          diffSchema(
            `${ctx} request`,
            baseRequestSchema,
            currRequestSchema,
            baseComponents,
            currComponents,
            findings,
          )
        }
      }

      const baseResponses = isRecord(baseOp.responses)
        ? (baseOp.responses as Record<string, JsonSchema>)
        : {}
      const currResponses = isRecord(currOp.responses)
        ? (currOp.responses as Record<string, JsonSchema>)
        : {}
      for (const [status, baseResponse] of Object.entries(baseResponses)) {
        const currResponse = currResponses[status]
        if (!currResponse) {
          findings.push({ operation: `${ctx} ${status}`, message: 'response status removed' })
          continue
        }
        const baseSchema = firstContentSchema(baseResponse)
        if (baseSchema) {
          diffSchema(
            `${ctx} ${status}`,
            baseSchema,
            firstContentSchema(currResponse),
            baseComponents,
            currComponents,
            findings,
          )
        }
      }
    }
  }

  return findings
}
