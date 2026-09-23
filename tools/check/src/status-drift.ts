/**
 * Check that the status codes an operation declares are the ones its handler
 * and the guards in front of it actually write.
 *
 * wi-382 closed five of these by hand and left the sweep undone; wi-386 found
 * that the disagreement is systemic rather than incidental — 401 was declared on
 * 10 operations and written by the admin guard on 250 of them. The rule the
 * contract now holds to lives in docs/design/application/api-guidelines.md, 「ステータスコードの宣言」.
 *
 * The chain is wi-385's: operationId to route to handler. What is new here is
 * the reading rule, and it is the whole design. Only a holder of the echo
 * context can write a response, so only those functions are read. Within them,
 * two kinds of helper have to be told apart.
 *
 * A **guard** decides from the request standing in front of it — an origin, a
 * CSRF token, a role, a rate limit. Every branch of a guard is reachable
 * wherever it is called, so following it is sound.
 *
 * An **error mapper** decides from an error value the use case produced. Which
 * branch runs is settled somewhere this reader cannot see: following
 * WriteAccountError makes every account operation look able to answer 409
 * `mfa_already_enrolled`, which is drift that is not there. A mapper is
 * therefore not followed, and the operation that calls one is recorded as
 * partially read — its missing declarations are still reported, because those
 * were read, but nothing it declares is called over-declared.
 */

import type { GoFile } from './contract-drift.ts'
import { collectRoutes } from './contract-drift.ts'

export type { GoFile }

/** `key` is `<rule> <operationId>`, so a finding reads the same after a rename. */
export type Finding = { key: string; operationId: string; message: string }

export type Unresolved = {
  operationId: string
  reason: 'route-not-found' | 'handler-not-found'
  detail: string
}

/** An operation whose writers were not all read, and the ones that stopped it. */
export type Unread = { operationId: string; writers: string[] }

export type StatusDriftResult = {
  findings: Finding[]
  unresolved: Unresolved[]
  unread: Unread[]
}

export type OpenAPIDocument = {
  paths?: Record<
    string,
    Record<string, { operationId?: string; responses?: Record<string, unknown> }>
  >
  components?: { schemas?: Record<string, unknown> }
}

/** A context-carrying function, with what it writes and what stopped the reader. */
export type Responder = {
  path: string
  statuses: Set<number>
  problemCodes: Map<number, Set<string>>
  unread: string[]
  /** 呼び出し先を 1 つに決められなかった前提判定。読み飛ばすと帰属が黙って欠ける。 */
  ambiguous: Ambiguity[]
  /** この名前を持つ定義の位置。2 つ以上あると、経路からどれを読むかを決められない。 */
  definitions: string[]
}

/** 字面から呼び出し先を 1 つに決められなかった呼び出しと、候補の定義位置。 */
export type Ambiguity = { name: string; candidates: string[] }

const HTTP_METHODS = ['get', 'post', 'put', 'patch', 'delete', 'head', 'options']

/**
 * net/http's status constants. Spelled out rather than derived: the checker must
 * fail to read a constant it does not know rather than guess a number for it.
 */
const STATUS: Record<string, number> = {
  StatusOK: 200,
  StatusCreated: 201,
  StatusAccepted: 202,
  StatusNoContent: 204,
  StatusMovedPermanently: 301,
  StatusFound: 302,
  StatusSeeOther: 303,
  StatusNotModified: 304,
  StatusTemporaryRedirect: 307,
  StatusPermanentRedirect: 308,
  StatusBadRequest: 400,
  StatusUnauthorized: 401,
  StatusForbidden: 403,
  StatusNotFound: 404,
  StatusMethodNotAllowed: 405,
  StatusNotAcceptable: 406,
  StatusConflict: 409,
  StatusGone: 410,
  StatusPreconditionFailed: 412,
  StatusRequestEntityTooLarge: 413,
  StatusUnsupportedMediaType: 415,
  StatusUnprocessableEntity: 422,
  StatusTooManyRequests: 429,
  StatusInternalServerError: 500,
  StatusNotImplemented: 501,
  StatusBadGateway: 502,
  StatusServiceUnavailable: 503,
  StatusGatewayTimeout: 504,
}

/**
 * The writers that take the status as an argument, so the caller settles it and
 * the callee never has to be read.
 *
 * echo's own methods, plus the two shared wrappers every context writes through.
 * `WriteProblem` and `NoStoreJSON` would be discovered anyway — they reach
 * `c.JSON` — but naming them keeps the reading rule the same whether or not
 * support_http happens to be among the files handed in.
 */
const STATUS_ARGUMENT = new Set([
  'WriteProblem',
  'NoStoreJSON',
  'JSON',
  'JSONBlob',
  'JSONPretty',
  'String',
  'NoContent',
  'Redirect',
  'Blob',
  'HTML',
  'HTMLBlob',
  'XML',
  'XMLBlob',
  'Stream',
  'NewHTTPError',
])

/**
 * Helpers that dispatch on an error value and are followed anyway, because the
 * error comes from the guard standing next to the call rather than from a use
 * case, so both branches are reachable at every call site.
 *
 * `WriteAdminAccessError` and `WriteAccessTokenError` answer the two outcomes of
 * `RequireAdmin` and of bearer validation: not authenticated (401) and not
 * permitted (403). `WriteServerError` is the shared 500 fallback written out
 * locally; following it costs nothing because 500 is not declared anywhere (see
 * PIPELINE below) and refusing to follow it would cost the operation its S2.
 */
const FOLLOWED = new Set(['WriteAdminAccessError', 'WriteAccessTokenError', 'WriteServerError'])

/**
 * 500 is the shared error handler's answer to an error no handler mapped. It is
 * reachable from every operation and means the same thing on all of them, so
 * docs/design/application/api-guidelines.md keeps it out of the per-operation declaration: it is neither
 * demanded when written nor reported when declared.
 */
const PIPELINE = new Set([500])

/**
 * Statuses `support_http.ErrorHandler` can produce from a bare `return err`.
 * Whether the error that triggers one arises is a property of the use case, not
 * of the handler's text, so S2 never calls these over-declared. S1 still reports
 * them where the handler itself writes one.
 */
const ERROR_HANDLER = new Set([401, 403, 422, 500])

/** The text between `source[open]` and its matching brace, braces excluded. */
function sliceBalanced(source: string, open: number, pair = '{}'): string | undefined {
  let depth = 0
  for (let i = open; i < source.length; i++) {
    if (source[i] === pair[0]) depth++
    else if (source[i] === pair[1]) {
      depth--
      if (depth === 0) return source.slice(open + 1, i)
    }
  }
  return undefined
}

/**
 * The brace that opens a function body, starting from the end of `func <name>`.
 *
 * The signature can carry braces of its own — `map[string]any`, `struct{}`,
 * `chan struct{}` — so the body brace is the first one outside parentheses and
 * brackets that is not opening a struct or interface type.
 */
function bodyBrace(source: string, from: number): number | undefined {
  let paren = 0
  let bracket = 0
  for (let i = from; i < source.length; i++) {
    const ch = source[i]
    if (ch === '(') paren++
    else if (ch === ')') paren--
    else if (ch === '[') bracket++
    else if (ch === ']') bracket--
    else if (ch === '{' && paren === 0 && bracket === 0) {
      const before = source.slice(Math.max(0, i - 12), i).trimEnd()
      if (!before.endsWith('struct') && !before.endsWith('interface')) return i
      const inner = sliceBalanced(source, i)
      if (inner === undefined) return undefined
      i += inner.length + 1
    } else if (ch === '\n' && paren === 0 && bracket === 0) {
      // A declaration with no body: an interface method, or a signature this
      // reader mis-split. Either way there is nothing to read.
      return undefined
    }
  }
  return undefined
}

type Definition = {
  name: string
  path: string
  /** Go ではディレクトリが 1 つのパッケージになるので、同じ名前の定義をこれで区別する。 */
  directory: string
  /** メソッドの受け手の変数名。関数と、受け手に名前のないメソッドでは undefined になる。 */
  receiver?: string
  body: string
  parameters: string
}

/**
 * Context-carrying function definitions, by name. A function that does not hold
 * the echo context cannot write a response, so a repository comparing an
 * upstream `resp.StatusCode` against `http.StatusNotFound` is never read as
 * writing a 404.
 */
function collectDefinitions(files: GoFile[]): Map<string, Definition[]> {
  const definitions = new Map<string, Definition[]>()
  for (const file of files) {
    for (const match of file.source.matchAll(/^func\s+(?:\(([^)]*)\)\s*)?(\w+)/gm)) {
      const name = match[2]
      if (!name) continue
      const open = bodyBrace(file.source, match.index + match[0].length)
      if (open === undefined) continue
      const body = sliceBalanced(file.source, open)
      if (body === undefined) continue
      const signature = file.source.slice(match.index, open)
      if (!signature.includes('*echo.Context')) continue
      const parametersAt = signature.indexOf('(', match[0].length - name.length)
      const parameters =
        parametersAt === -1 ? '' : (sliceBalanced(signature, parametersAt, '()') ?? '')
      definitions.set(name, [
        ...(definitions.get(name) ?? []),
        {
          name,
          path: file.path,
          directory: file.path.slice(0, Math.max(0, file.path.lastIndexOf('/'))),
          receiver: match[1] === undefined ? undefined : /^\s*(\w+)\s+\*?\w/.exec(match[1])?.[1],
          body,
          parameters,
        },
      ])
    }
  }
  return definitions
}

/**
 * 本体に書かれた呼び出しと、括弧の対応で切り出した引数。
 *
 * `qualifier` は名前の前の `.` までの字面である。`d.Auth.requireAdmin(` なら
 * `d.Auth`、修飾のない `requireAdmin(` なら空文字、`c.Request().Context(` の
 * ように式の結果を受け手にする呼び出しなら `.` になる。
 */
type CallSite = { name: string; qualifier: string; args: string }

function callSites(body: string): CallSite[] {
  const sites: CallSite[] = []
  for (const match of body.matchAll(/([\w.]*\.)?(\w+)\s*\(/g)) {
    const name = match[2]
    if (!name) continue
    const open = match.index + match[0].length - 1
    const qualifier = match[1] === undefined ? '' : match[1].slice(0, -1) || '.'
    sites.push({ name, qualifier, args: sliceBalanced(body, open, '()') ?? '' })
  }
  return sites
}

/**
 * 呼び出しが届きうる定義。
 *
 * 修飾のない呼び出しは、同じパッケージの関数にしか届かない。呼び出し側の受け手の
 * 変数を通した呼び出しは、同じパッケージの定義を優先する。それ以外の修飾
 * （フィールド、パッケージ名、式の結果）は受け手の型を字面から決められないので、
 * 同じ名前の定義がすべて候補に残る。
 */
function candidatesFor(
  call: CallSite,
  caller: Definition,
  definitions: Map<string, Definition[]>,
): Definition[] {
  const named = definitions.get(call.name) ?? []
  const local = named.filter((definition) => definition.directory === caller.directory)
  if (call.qualifier === '') return local
  if (call.qualifier === caller.receiver && local.length > 0) return local
  return named
}

function statusesIn(args: string): number[] {
  const found: number[] = []
  for (const match of args.matchAll(/http\.(Status\w+)/g)) {
    const code = STATUS[match[1] ?? '']
    if (code) found.push(code)
  }
  return found
}

/** Literal Problem Details codes whose status is settled at a WriteProblem call. */
function problemCodesIn(call: { name: string; args: string }): Array<[number, string]> {
  if (call.name !== 'WriteProblem') return []
  const found: Array<[number, string]> = []
  for (const match of call.args.matchAll(/http\.(Status\w+)\s*,\s*"([a-z0-9_]+)"/g)) {
    const status = STATUS[match[1] ?? '']
    const code = match[2]
    if (status && code) found.push([status, code])
  }
  return found
}

/**
 * A guard decides from the request; a mapper decides from an error handed to it.
 * Taking an `error` parameter is what that looks like, and so is dispatching on
 * one — provisioning's `writeError` switches on `isNotFound(err)` rather than on
 * `errors.Is`, and is a mapper all the same.
 */
function isGuard(definition: Definition): boolean {
  if (FOLLOWED.has(definition.name)) return true
  return (
    !/(^|[\s,([])error([\s,)]|$)/.test(definition.parameters) &&
    !/\berrors\.(Is|As|AsType)\b/.test(definition.body)
  )
}

/**
 * Every context-carrying function, with the statuses it writes and the writers
 * that stopped the reader.
 *
 * A call settled by a constant at the call site — `WriteProblem(c,
 * http.StatusNotFound, ...)` — needs no reading of the callee. Only a call that
 * hands over the decision does, and that is where the guard/mapper split
 * applies.
 */
export function collectResponders(files: GoFile[]): Map<string, Responder> {
  const definitions = collectDefinitions(files)

  // 呼び出しの解決は定義ごとに一度だけ行う。同じ名前でも、呼び出し側の
  // パッケージと受け手によって届く定義が変わるためである。
  const sites = new Map<Definition, Array<CallSite & { candidates: Definition[] }>>()
  for (const found of definitions.values()) {
    for (const definition of found) {
      sites.set(
        definition,
        callSites(definition.body).map((call) => ({
          ...call,
          candidates: candidatesFor(call, definition, definitions),
        })),
      )
    }
  }

  // A definition is a writer when it reaches a response write. Grown from
  // echo's primitives until it stops changing, so a call to something that
  // never writes is not mistaken for a silent one that does.
  const writers = new Set<Definition>()
  const writes = (call: CallSite & { candidates: Definition[] }) =>
    STATUS_ARGUMENT.has(call.name) || call.candidates.some((callee) => writers.has(callee))
  for (;;) {
    let grew = false
    for (const [definition, calls] of sites) {
      if (writers.has(definition)) continue
      if (calls.some(writes)) {
        writers.add(definition)
        grew = true
      }
    }
    if (!grew) break
  }

  const read = new Map<Definition, Responder>()
  const delegates = new Map<Definition, Definition[]>()
  const callers = new Map<Definition, Set<Definition>>()
  for (const [definition, calls] of sites) {
    const statuses = new Set<number>()
    const problemCodes = new Map<number, Set<string>>()
    const unread = new Set<string>()
    const ambiguous: Ambiguity[] = []
    const followed = new Set<Definition>()
    for (const call of calls) {
      for (const [status, code] of problemCodesIn(call)) {
        problemCodes.set(status, (problemCodes.get(status) ?? new Set()).add(code))
      }
      // Handing the raw response to something else — `ServeHTTP(c.Response(),
      // ...)` on /metrics — writes a status this reader never sees. Reading the
      // header off the context is not that: there the response object is the
      // receiver, not an argument.
      if (call.args.includes('c.Response()')) {
        unread.add(call.name)
        continue
      }
      if (!writes(call)) continue
      const constants = statusesIn(call.args)
      if (constants.length > 0) {
        for (const status of constants) statuses.add(status)
        continue
      }
      if (STATUS_ARGUMENT.has(call.name)) {
        unread.add(call.name)
        continue
      }
      const callees = call.candidates.filter((callee) => callee !== definition)
      const [callee] = callees
      if (callee === undefined) continue
      if (callees.length === 1) {
        if (isGuard(callee)) followed.add(callee)
        else unread.add(call.name)
        continue
      }
      // 候補がすべてエラー値で分岐するなら、1 つに決まっても読まないので結果は変わらない。
      // 前提判定が候補に入るときは、どれを読むかで帰属する状態コードが変わる。
      if (callees.some(isGuard)) {
        ambiguous.push({ name: call.name, candidates: callees.map((found) => found.path) })
      } else {
        unread.add(call.name)
      }
    }
    read.set(definition, {
      path: definition.path,
      statuses,
      problemCodes,
      unread: [...unread],
      ambiguous,
      definitions: (definitions.get(definition.name) ?? []).map((found) => found.path),
    })
    delegates.set(definition, [...followed])
    for (const callee of followed) {
      callers.set(callee, (callers.get(callee) ?? new Set()).add(definition))
    }
  }

  // A guard's statuses and its own unread writers belong to everything that
  // stands behind it. Iterated to a fixed point so a chain of thin guards —
  // requireWorkflowAdmin to requireAdmin to WriteAdminAccessError — carries all
  // the way through, and so a cycle terminates.
  const size = (responder: Responder) =>
    responder.statuses.size +
    responder.unread.length +
    responder.ambiguous.length +
    [...responder.problemCodes.values()].reduce((sum, codes) => sum + codes.size, 0)
  const work = [...read.keys()]
  while (work.length > 0) {
    const definition = work.pop() as Definition
    const responder = read.get(definition) as Responder
    const before = size(responder)
    for (const callee of delegates.get(definition) ?? []) {
      const target = read.get(callee)
      if (!target) continue
      for (const status of target.statuses) responder.statuses.add(status)
      for (const [status, codes] of target.problemCodes) {
        const own = responder.problemCodes.get(status) ?? new Set<string>()
        for (const code of codes) own.add(code)
        responder.problemCodes.set(status, own)
      }
      for (const writer of target.unread) {
        if (!responder.unread.includes(writer)) responder.unread.push(writer)
      }
      for (const entry of target.ambiguous) {
        if (!responder.ambiguous.some((own) => own.name === entry.name)) {
          responder.ambiguous.push(entry)
        }
      }
    }
    if (size(responder) !== before) {
      for (const caller of callers.get(definition) ?? []) work.push(caller)
    }
  }

  // 経路の登録はハンドラーの名前しか持たないので、呼び出し側へは名前で渡す。
  // 同じ名前の定義が 2 つ以上あれば、どれを読んだ結果も渡さない。
  const responders = new Map<string, Responder>()
  for (const [name, found] of definitions) {
    const [only] = found
    const responder = found.length === 1 && only ? read.get(only) : undefined
    responders.set(
      name,
      responder ?? {
        path: '',
        statuses: new Set(),
        problemCodes: new Map(),
        unread: [],
        ambiguous: [],
        definitions: found.map((definition) => definition.path),
      },
    )
  }
  return responders
}

/** `/things/:id` and `/things/{id}` are the same route written twice. */
function pathShape(key: string): string {
  return key.replace(/\{[^}]*\}/g, '{}')
}

const sortNumbers = (values: Iterable<number>) => [...values].sort((a, b) => a - b)

function collectDeclaredProblemCodes(
  node: unknown,
  document: OpenAPIDocument,
  codes = new Set<string>(),
  seenRefs = new Set<string>(),
): Set<string> {
  if (Array.isArray(node)) {
    for (const value of node) collectDeclaredProblemCodes(value, document, codes, seenRefs)
    return codes
  }
  if (!node || typeof node !== 'object') return codes

  const record = node as Record<string, unknown>
  const ref = record.$ref
  if (typeof ref === 'string' && ref.startsWith('#/components/schemas/') && !seenRefs.has(ref)) {
    seenRefs.add(ref)
    const name = ref.slice('#/components/schemas/'.length)
    collectDeclaredProblemCodes(document.components?.schemas?.[name], document, codes, seenRefs)
  }
  const description = record.description
  if (typeof description === 'string') {
    for (const match of description.matchAll(/urn:idmagic:error:([a-z0-9_]+)/g)) {
      const code = match[1]
      if (code) codes.add(code)
    }
  }
  for (const [key, value] of Object.entries(record)) {
    if (key !== '$ref' && key !== 'description') {
      collectDeclaredProblemCodes(value, document, codes, seenRefs)
    }
  }
  return codes
}

function declared403ProblemCodes(
  response: unknown,
  document: OpenAPIDocument,
): Set<string> | undefined {
  if (!response || typeof response !== 'object') return undefined
  const content = (response as Record<string, unknown>).content
  if (!content || typeof content !== 'object') return undefined
  const problem = (content as Record<string, unknown>)['application/problem+json']
  if (!problem || typeof problem !== 'object') return undefined
  const schema = (problem as Record<string, unknown>).schema
  if (!schema) return undefined
  return collectDeclaredProblemCodes(schema, document)
}

/**
 * Compare every operation the contract declares against the statuses the Go that
 * serves it writes.
 */
export function diffStatusCodes(document: OpenAPIDocument, goFiles: GoFile[]): StatusDriftResult {
  const routes = collectRoutes(goFiles)
  const responders = collectResponders(goFiles)
  const shapes = new Map<string, string[]>()
  for (const [key, handler] of routes) {
    const shape = pathShape(key)
    shapes.set(shape, [...(shapes.get(shape) ?? []), handler])
  }

  const findings: Finding[] = []
  const unresolved: Unresolved[] = []
  const unread: Unread[] = []

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
      const responder = responders.get(handlerName)
      if (!responder) {
        unresolved.push({ operationId, reason: 'handler-not-found', detail: handlerName })
        continue
      }
      // Two functions of one name: the statuses read are the union of both, and
      // attributing another handler's 404 to this operation is exactly the
      // silent wrong answer this check exists to avoid. 件数に数えるだけでは
      // 検査が通ってしまうので、読めなかったことを失敗として報告する。
      if (responder.definitions.length > 1) {
        findings.push({
          key: `A1 ${operationId}`,
          operationId,
          message:
            `A1 ${operationId}: ${handlerName} is defined at ${responder.definitions.join(', ')}; ` +
            'the reader cannot tell which one the route registers',
        })
        continue
      }
      if (responder.ambiguous.length > 0) {
        const calls = responder.ambiguous
          .map((entry) => `${entry.name}, which is defined at ${entry.candidates.join(', ')}`)
          .join('; ')
        findings.push({
          key: `A1 ${operationId}`,
          operationId,
          message:
            `A1 ${operationId}: ${handlerName} calls ${calls}; ` +
            'the reader cannot tell which one runs',
        })
      }

      const declared = new Set(
        Object.keys(operation.responses ?? {})
          .map(Number)
          .filter((status) => Number.isFinite(status)),
      )
      const written = responder.statuses

      const missing = sortNumbers(written).filter(
        (status) => !declared.has(status) && !PIPELINE.has(status),
      )
      if (missing.length > 0) {
        findings.push({
          key: `S1 ${operationId}`,
          operationId,
          message:
            `S1 ${operationId}: ${handlerName} writes ${missing.join(', ')}, ` +
            'which the contract does not declare',
        })
      }

      const response403 = operation.responses?.['403']
      if (response403) {
        const declaredCodes = declared403ProblemCodes(response403, document)
        if (declaredCodes) {
          const missingCodes = [...(responder.problemCodes.get(403) ?? [])]
            .filter((code) => !declaredCodes.has(code))
            .sort()
          if (missingCodes.length > 0) {
            findings.push({
              key: `B1 ${operationId}`,
              operationId,
              message:
                `B1 ${operationId}: ${handlerName} can write 403 Problem Details code ` +
                `${missingCodes.join(', ')}, which the declared 403 body does not include`,
            })
          }
        }
      }

      if (responder.unread.length > 0) {
        unread.push({ operationId, writers: responder.unread })
        continue
      }
      if (responder.ambiguous.length > 0) continue

      const extra = sortNumbers(declared).filter(
        (status) => !written.has(status) && !ERROR_HANDLER.has(status),
      )
      if (extra.length > 0) {
        findings.push({
          key: `S2 ${operationId}`,
          operationId,
          message:
            `S2 ${operationId}: the contract declares ${extra.join(', ')}, ` +
            `which ${handlerName} never writes`,
        })
      }
    }
  }
  return { findings, unresolved, unread }
}
