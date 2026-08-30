/**
 * Check that what TypeSpec declares as an operation's request and response body
 * is what the Go handler actually decodes and writes.
 *
 * wi-381 found that `UserAttributeDef` declared 2 of its 10 fields; wi-382 found
 * the same shape spread across the contract, including request bodies wrapped in
 * envelopes the server never accepted. Both were found by a person reading
 * handlers one at a time, which is not a thing that happens twice.
 *
 * The chain walked here is the one that work did by hand: operationId to route
 * to handler to the struct it decodes into. It is regex over Go source, the same
 * technique security-controls.ts uses, and it therefore cannot follow everything
 * — a response held in a variable needs type inference this does not do. What it
 * cannot follow is reported as unresolved rather than counted as agreement,
 * because a check that silently passes what it did not read is worse than no
 * check: it answers a question it never asked.
 */

export type GoFile = { path: string; source: string }

/**
 * `key` is `<rule> <operationId>`, which names the drift independently of the Go
 * handler it was found through, so a rename does not change how a finding reads.
 */
export type Finding = { key: string; operationId: string; message: string }

export type UnresolvedReason =
  | 'route-not-found'
  | 'handler-not-found'
  | 'request-type-not-found'
  | 'request-struct-not-found'
  | 'response-not-a-literal'

export type Unresolved = { operationId: string; reason: UnresolvedReason; detail: string }

export type DriftResult = { findings: Finding[]; unresolved: Unresolved[] }

export type OpenAPIDocument = {
  paths?: Record<string, Record<string, OpenAPIOperation>>
  components?: { schemas?: Record<string, OpenAPISchema> }
}

type OpenAPIOperation = {
  operationId?: string
  parameters?: Array<{ name?: string; in?: string }>
  requestBody?: { content?: Record<string, { schema?: OpenAPISchema }> }
  responses?: Record<string, { content?: Record<string, { schema?: OpenAPISchema }> }>
}

type OpenAPISchema = {
  $ref?: string
  type?: string
  properties?: Record<string, unknown>
  allOf?: OpenAPISchema[]
}

const HTTP_METHODS = ['get', 'post', 'put', 'patch', 'delete', 'head', 'options']

/**
 * Route registrations, keyed `METHOD /path` with the path in the OpenAPI
 * `{param}` form. echo writes `:param`, so the two are normalized here rather
 * than at every comparison.
 */
export function collectRoutes(files: GoFile[]): Map<string, string> {
  const routes = new Map<string, string>()
  // Two registration forms are in use. The direct one, `g.POST("/path",
  // d.handleX)`, and a closure that delegates, `g.GET("/path", func(c
  // *echo.Context) error { return handleX(d, c) })` — the Authentication package
  // registers most of its account routes the second way, and reading only the
  // first left a third of the contract unresolved.
  //
  // The receiver is dropped from the direct form: handler names are unique
  // enough here, and keeping it would tie the check to how each package names
  // its Deps value.
  const direct =
    /\.(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\(\s*"([^"]+)"\s*,\s*(?:[\w.]+\.)?(\w+)\s*[,)]/g
  const delegating =
    /\.(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\(\s*"([^"]+)"\s*,\s*func\s*\([^)]*\)\s*error\s*\{\s*return\s+(?:[\w.]+\.)?(\w+)\s*\(/g
  for (const file of files) {
    for (const pattern of [direct, delegating]) {
      for (const match of file.source.matchAll(pattern)) {
        const [, method, path, handler] = match
        if (!method || !path || !handler) continue
        const key = `${method} ${normalizePath(path)}`
        // The direct pattern also matches the `func` keyword of a delegating
        // registration, so the delegating pass runs second and wins.
        if (pattern === delegating || !routes.has(key)) routes.set(key, handler)
      }
    }
  }
  return routes
}

/** `/things/:id` and `/things/{id}` are the same route written twice. */
function normalizePath(path: string): string {
  return path.replace(/:([A-Za-z_][\w]*)/g, '{$1}')
}

/**
 * The route with its parameter names replaced by position. The contract and the
 * code disagree about some of these names — `{tenant_id}` against
 * `:target_tenant_id` — and refusing to compare those operations would hide a
 * body drift behind a naming difference. Used only as a fallback, and only when
 * it names exactly one handler.
 */
function pathShape(key: string): string {
  return key.replace(/\{[^}]*\}/g, '{}')
}

function shapeIndex(routes: Map<string, string>): Map<string, string[]> {
  const index = new Map<string, string[]>()
  for (const [key, handler] of routes) {
    const shape = pathShape(key)
    index.set(shape, [...(index.get(shape) ?? []), handler])
  }
  return index
}

export type GoHandler = {
  path: string
  body: string
  /** The struct the handler decodes the request body into, when it is followable. */
  requestType?: string
  /** The keys of a map literal written directly in the response call. */
  responseKeys?: string[]
  /** The struct type of a composite literal written directly in the response call. */
  responseType?: string
}

/**
 * Handler bodies, cut at the brace that matches the opening one. Cutting at the
 * first `\n}` instead would merge a handler with whatever follows it, which is
 * how a check like this quietly starts reading the wrong function.
 */
export function collectGoHandlers(files: GoFile[]): Map<string, GoHandler> {
  const handlers = new Map<string, GoHandler>()
  // Both `func (d Deps) handleX(c *echo.Context) error` and
  // `func handleX(d Deps, c *echo.Context) error`. The latter is what the
  // delegating registrations call.
  const pattern =
    /func\s+(?:\([^)]*\)\s*)?(\w+)\s*\((?:[^)]*,\s*)?c\s+\*echo\.Context\s*\)\s*error\s*\{/g
  for (const file of files) {
    for (const match of file.source.matchAll(pattern)) {
      const name = match[1]
      if (!name) continue
      const openIndex = match.index + match[0].length - 1
      const body = sliceBalanced(file.source, openIndex)
      if (body === undefined) continue
      handlers.set(name, { path: file.path, body, ...readBodyShapes(body) })
    }
  }
  return handlers
}

/** The text between `source[open]` and its matching brace, braces excluded. */
function sliceBalanced(source: string, open: number): string | undefined {
  let depth = 0
  for (let i = open; i < source.length; i++) {
    const ch = source[i]
    if (ch === '{') depth++
    else if (ch === '}') {
      depth--
      if (depth === 0) return source.slice(open + 1, i)
    }
  }
  return undefined
}

function readBodyShapes(body: string): Omit<GoHandler, 'path' | 'body'> {
  return {
    requestType: readRequestType(body),
    ...readResponseShape(body),
  }
}

/**
 * The struct a handler decodes into. Both spellings in this repository take the
 * address of a local declared in the same body, so the local's declaration is
 * what names the type. A decode into a field or a parameter is left unresolved.
 */
function readRequestType(body: string): string | undefined {
  const decode = body.match(/(?:DecodeJSON\([^,]+,\s*&|\.Bind\(\s*&)(\w+)\s*\)/)
  const local = decode?.[1]
  if (!local) return undefined
  const declaration = body.match(new RegExp(`var\\s+${local}\\s+([\\w.]+)`))
  return declaration?.[1]
}

/** The 2xx statuses a handler writes a success body with. */
const SUCCESS_STATUS = /^http\.Status(OK|Created|Accepted|NonAuthoritativeInfo|NoContent)$/

/**
 * The success response body's first-level shape.
 *
 * Only 2xx writes are read: a handler commonly writes an error body before its
 * success one, and comparing the contract's success body against the first write
 * in the file reports a difference that is not there. When two success writes
 * disagree — `/readyz` answers differently with and without `verbose` — the
 * shape is left unresolved rather than picked, because choosing between them
 * would make the finding depend on which branch the reader happened to see.
 *
 * A variable is not followed either: naming its type needs inference this does
 * not do, and guessing would be the silent-wrong-answer this check exists to
 * avoid.
 */
function readResponseShape(body: string): { responseKeys?: string[]; responseType?: string } {
  const shapes: Array<{ responseKeys?: string[]; responseType?: string }> = []
  const call = /(?:NoStoreJSON|\.JSON)\(\s*(?:c\s*,\s*)?([\w.]+)\s*,\s*/g
  for (const match of body.matchAll(call)) {
    if (!SUCCESS_STATUS.test(match[1] ?? '')) continue
    const rest = body.slice(match.index + match[0].length)

    // Any element type: the keys are what the contract compares, and reading
    // only `map[string]any` left every `map[string]string` response unchecked.
    const mapLiteral = rest.match(/^map\[string\](?:interface\{\}|[\w.[\]*]+)\{/)
    if (mapLiteral) {
      const inner = sliceBalanced(rest, mapLiteral[0].length - 1)
      if (inner === undefined) return {}
      shapes.push({ responseKeys: topLevelMapKeys(inner) })
      continue
    }
    const structLiteral = rest.match(/^([A-Za-z_]\w*)\{/)
    if (structLiteral?.[1]) {
      shapes.push({ responseType: structLiteral[1] })
      continue
    }
    // A success write whose body is a variable makes the whole handler
    // unresolved: the shapes that follow cannot be assumed to be the only ones.
    return {}
  }
  if (shapes.length === 0) return {}
  const first = JSON.stringify(shapes[0])
  return shapes.every((shape) => JSON.stringify(shape) === first) ? (shapes[0] as object) : {}
}

/**
 * The keys of a map literal's own level, ignoring anything nested inside it.
 *
 * The contract's response body is compared one level deep, so a key belonging to
 * a nested map is not a response key. Matching every quoted name reported five
 * spurious extras on CheckAccess alone, which is how a checker like this loses
 * the reader's trust.
 */
function topLevelMapKeys(inner: string): string[] {
  const keys: string[] = []
  let depth = 0
  for (let i = 0; i < inner.length; i++) {
    const ch = inner[i]
    if (ch === '{' || ch === '[' || ch === '(') depth++
    else if (ch === '}' || ch === ']' || ch === ')') depth--
    else if (ch === '"') {
      const end = inner.indexOf('"', i + 1)
      if (end === -1) break
      const literal = inner.slice(i + 1, end)
      // A key is followed by a colon; a value is not.
      if (depth === 0 && /^\s*:/.test(inner.slice(end + 1))) keys.push(literal)
      i = end
    }
  }
  return keys
}

/**
 * Struct name to the json tag names of its fields, `-` and omitempty removed.
 *
 * Kept as the simple shape because most callers only compare key sets;
 * collectGoStructFields carries the field types the nested comparison needs.
 */
export function collectGoStructs(files: GoFile[]): Map<string, string[]> {
  const structs = new Map<string, string[]>()
  for (const [name, fields] of collectGoStructFields(files)) {
    structs.set(name, [...fields.keys()])
  }
  return structs
}

/**
 * Struct name to a map of json tag name -> the field's Go type, with slice,
 * pointer and package qualifiers stripped. The type is what lets a nested
 * contract model be compared against the struct that actually decodes it:
 * wi-381's defect was in `UserAttributeDef`, which is nested inside another
 * model rather than being a request body of its own.
 */
export function collectGoStructFields(files: GoFile[]): Map<string, Map<string, string>> {
  const structs = new Map<string, Map<string, string>>()
  const pattern = /type\s+(\w+)\s+struct\s*\{/g
  for (const file of files) {
    for (const match of file.source.matchAll(pattern)) {
      const name = match[1]
      if (!name) continue
      const body = sliceBalanced(file.source, match.index + match[0].length - 1)
      if (body === undefined) continue
      const fields = new Map<string, string>()
      // `Name  Type  `json:"tag"`` on one line, which is what gofumpt produces.
      for (const field of body.matchAll(/^\s*\w+\s+([^`\n]+?)\s*`[^`]*json:"([^"]*)"/gm)) {
        const tag = (field[2] ?? '').split(',')[0] ?? ''
        if (tag === '' || tag === '-') continue
        const type = (field[1] ?? '')
          .replace(/[[\]*]/g, '')
          .replace(/^.*\./, '')
          .trim()
        fields.set(tag, type)
      }
      structs.set(name, fields)
    }
  }
  return structs
}

/** Resolve a schema through `$ref` and flatten a single-level `allOf`. */
function resolveSchema(
  schema: OpenAPISchema | undefined,
  document: OpenAPIDocument,
  seen = new Set<string>(),
): OpenAPISchema | undefined {
  if (!schema) return undefined
  if (schema.$ref) {
    const name = schema.$ref.replace('#/components/schemas/', '')
    if (seen.has(name)) return undefined
    seen.add(name)
    return resolveSchema(document.components?.schemas?.[name], document, seen)
  }
  if (schema.allOf) {
    const properties: Record<string, unknown> = {}
    for (const part of schema.allOf) {
      Object.assign(properties, resolveSchema(part, document, seen)?.properties ?? {})
    }
    return { type: 'object', properties }
  }
  return schema
}

function jsonSchemaOf(
  content: Record<string, { schema?: OpenAPISchema }> | undefined,
): OpenAPISchema | undefined {
  if (!content) return undefined
  for (const [mediaType, entry] of Object.entries(content)) {
    if (mediaType.includes('json')) return entry.schema
  }
  return undefined
}

/**
 * The success response's schema. Only 2xx is compared: an error body is written
 * by a shared writer rather than by the handler, so the handler's own text says
 * nothing about it.
 */
function successResponseSchema(
  operation: OpenAPIOperation,
  document: OpenAPIDocument,
): OpenAPISchema | undefined {
  for (const [status, response] of Object.entries(operation.responses ?? {})) {
    if (!/^2\d\d$/.test(status)) continue
    const schema = resolveSchema(jsonSchemaOf(response.content), document)
    if (schema?.properties) return schema
  }
  return undefined
}

const symmetricDifference = (a: string[], b: string[]) => ({
  onlyInA: a.filter((x) => !b.includes(x)).sort(),
  onlyInB: b.filter((x) => !a.includes(x)).sort(),
})

/**
 * Compare one contract object against the Go struct that decodes it, then
 * descend into each property both sides agree exists and that is an object on
 * the contract side.
 *
 * Descending matters because the defect that motivated this check lived one
 * level down: `UserAttributeDef` declared 2 of its 10 fields while sitting
 * inside another model. A property missing on either side is reported here and
 * not descended into — the drift is already named, and going deeper would
 * report it twice. `seen` stops a self-referential model from looping.
 */
function compareRequestObject(
  schema: OpenAPISchema | undefined,
  structName: string,
  displayType: string,
  prefix: string,
  operationId: string,
  document: OpenAPIDocument,
  structFields: Map<string, Map<string, string>>,
  findings: Finding[],
  seen: Set<string>,
): void {
  if (seen.has(`${structName}@${prefix}`)) return
  seen.add(`${structName}@${prefix}`)

  const declared = Object.keys(schema?.properties ?? {})
  const fields = structFields.get(structName)
  if (!fields) return
  const at = prefix === '' ? '' : ` at ${prefix}`
  const { onlyInA, onlyInB } = symmetricDifference(declared, [...fields.keys()])
  if (onlyInA.length > 0) {
    findings.push({
      key: `D1 ${operationId}`,
      operationId,
      message: `D1 ${operationId}: the contract declares request properties ${displayType} does not decode${at}: ${onlyInA.join(', ')}`,
    })
  }
  if (onlyInB.length > 0) {
    findings.push({
      key: `D1 ${operationId}`,
      operationId,
      message: `D1 ${operationId}: ${displayType} decodes request properties the contract does not declare${at}: ${onlyInB.join(', ')}`,
    })
  }

  for (const [name, property] of Object.entries(schema?.properties ?? {})) {
    if (onlyInA.includes(name)) continue
    const nested = resolveSchema(unwrapArray(property as OpenAPISchema), document)
    if (!nested?.properties) continue
    const fieldType = fields.get(name)
    if (!fieldType || !structFields.has(fieldType)) continue
    compareRequestObject(
      nested,
      fieldType,
      fieldType,
      prefix === '' ? name : `${prefix}.${name}`,
      operationId,
      document,
      structFields,
      findings,
      seen,
    )
  }
}

/** `Thing[]` and `Thing` carry the same object shape for this comparison. */
function unwrapArray(schema: OpenAPISchema): OpenAPISchema {
  const items = (schema as { items?: OpenAPISchema }).items
  return schema.type === 'array' && items ? items : schema
}

/**
 * Compare every operation the contract declares against the Go that serves it.
 *
 * `routes` maps `METHOD /path` to a handler name; `goFiles` supplies the handler
 * bodies and struct definitions. They are separate parameters because the route
 * table is built once for the whole tree while the bodies are read per file.
 */
export function diffContract(
  document: OpenAPIDocument,
  routes: Map<string, string>,
  goFiles: GoFile[],
): DriftResult {
  const handlers = collectGoHandlers(goFiles)
  const structFields = collectGoStructFields(goFiles)
  const structs = collectGoStructs(goFiles)
  const shapes = shapeIndex(routes)
  const findings: Finding[] = []
  const unresolved: Unresolved[] = []

  for (const [path, pathItem] of Object.entries(document.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!HTTP_METHODS.includes(method)) continue
      const operationId = operation.operationId
      if (!operationId) continue
      const key = `${method.toUpperCase()} ${path}`

      const byShape = shapes.get(pathShape(key)) ?? []
      const handlerName = routes.get(key) ?? (byShape.length === 1 ? byShape[0] : undefined)
      if (!handlerName) {
        unresolved.push({ operationId, reason: 'route-not-found', detail: key })
        continue
      }
      const handler = handlers.get(handlerName)
      if (!handler) {
        unresolved.push({ operationId, reason: 'handler-not-found', detail: handlerName })
        continue
      }

      const requestSchema = resolveSchema(jsonSchemaOf(operation.requestBody?.content), document)
      const declaredProperties = Object.keys(requestSchema?.properties ?? {})

      if (declaredProperties.length > 0) {
        if (!handler.requestType) {
          unresolved.push({
            operationId,
            reason: 'request-type-not-found',
            detail: `${handler.path}:${handlerName}`,
          })
        } else {
          const structName = handler.requestType.replace(/^.*\./, '')
          if (!structFields.has(structName)) {
            unresolved.push({
              operationId,
              reason: 'request-struct-not-found',
              detail: handler.requestType,
            })
          } else {
            compareRequestObject(
              requestSchema,
              structName,
              handler.requestType,
              '',
              operationId,
              document,
              structFields,
              findings,
              new Set(),
            )
          }
        }

        // D2 needs only the contract: a name carried by the path or the query
        // string and repeated in the body is declared twice however the handler
        // reads it.
        const parameterNames = (operation.parameters ?? [])
          .filter((p) => p.in === 'path' || p.in === 'query')
          .map((p) => p.name ?? '')
        const duplicated = parameterNames.filter((name) => declaredProperties.includes(name)).sort()
        if (duplicated.length > 0) {
          findings.push({
            key: `D2 ${operationId}`,
            operationId,
            message: `D2 ${operationId}: ${duplicated.join(', ')} declared both as a path/query parameter and as a request body property`,
          })
        }
      }

      const responseSchema = successResponseSchema(operation, document)
      if (responseSchema) {
        const declaredKeys = Object.keys(responseSchema.properties ?? {})
        const actualKeys =
          handler.responseKeys ??
          (handler.responseType ? structs.get(handler.responseType) : undefined)
        if (!actualKeys) {
          unresolved.push({
            operationId,
            reason: 'response-not-a-literal',
            detail: `${handler.path}:${handlerName}`,
          })
        } else {
          const { onlyInA, onlyInB } = symmetricDifference(declaredKeys, actualKeys)
          if (onlyInA.length > 0) {
            findings.push({
              key: `D3 ${operationId}`,
              operationId,
              message: `D3 ${operationId}: the contract declares response keys the handler does not write: ${onlyInA.join(', ')}`,
            })
          }
          if (onlyInB.length > 0) {
            findings.push({
              key: `D3 ${operationId}`,
              operationId,
              message: `D3 ${operationId}: the handler writes response keys the contract does not declare: ${onlyInB.join(', ')}`,
            })
          }
        }
      }
    }
  }
  return { findings, unresolved }
}
