/**
 * Check that the status codes an operation declares are the ones its handler
 * and the guards in front of it actually write.
 *
 * wi-382 closed five of these by hand and left the sweep undone; wi-386 found
 * that the disagreement is systemic rather than incidental — 401 was declared on
 * 10 operations and written by the admin guard on 250 of them. The rule the
 * contract now holds to lives in docs/api-rules.md, Declared status codes.
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
  reason: 'route-not-found' | 'handler-not-found' | 'handler-ambiguous'
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
}

/** A context-carrying function, with what it writes and what stopped the reader. */
export type Responder = {
  path: string
  statuses: Set<number>
  unread: string[]
  /** How many definitions carry this name. More than one and none is read. */
  definitions: number
}

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
 * docs/api-rules.md keeps it out of the per-operation declaration: it is neither
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

type Definition = { path: string; body: string; parameters: string }

/**
 * Context-carrying function definitions, by name. A function that does not hold
 * the echo context cannot write a response, so a repository comparing an
 * upstream `resp.StatusCode` against `http.StatusNotFound` is never read as
 * writing a 404.
 */
function collectDefinitions(files: GoFile[]): Map<string, Definition[]> {
  const definitions = new Map<string, Definition[]>()
  for (const file of files) {
    for (const match of file.source.matchAll(/^func\s+(?:\([^)]*\)\s*)?(\w+)/gm)) {
      const name = match[1]
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
        { path: file.path, body, parameters },
      ])
    }
  }
  return definitions
}

/** Calls written in a body, each with its balanced argument text. */
function callSites(body: string): Array<{ name: string; args: string }> {
  const sites: Array<{ name: string; args: string }> = []
  for (const match of body.matchAll(/(?:\.|\b)(\w+)\s*\(/g)) {
    const name = match[1]
    if (!name) continue
    const open = match.index + match[0].length - 1
    sites.push({ name, args: sliceBalanced(body, open, '()') ?? '' })
  }
  return sites
}

function statusesIn(args: string): number[] {
  const found: number[] = []
  for (const match of args.matchAll(/http\.(Status\w+)/g)) {
    const code = STATUS[match[1] ?? '']
    if (code) found.push(code)
  }
  return found
}

/**
 * A guard decides from the request; a mapper decides from an error handed to it.
 * Taking an `error` parameter is what that looks like, and so is dispatching on
 * one — provisioning's `writeError` switches on `isNotFound(err)` rather than on
 * `errors.Is`, and is a mapper all the same.
 */
function isGuard(name: string, definitions: Map<string, Definition[]>): boolean {
  if (FOLLOWED.has(name)) return true
  const found = definitions.get(name) ?? []
  if (found.length === 0) return false
  return found.every(
    (definition) =>
      !/(^|[\s,([])error([\s,)]|$)/.test(definition.parameters) &&
      !/\berrors\.(Is|As|AsType)\b/.test(definition.body),
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

  // A name is a writer when it reaches a response write. Seeded with echo's
  // primitives and grown until it stops changing, so a call to something that
  // never writes is not mistaken for a silent one that does.
  const sites = new Map<string, Array<{ name: string; args: string }>>()
  for (const [name, found] of definitions) {
    sites.set(
      name,
      found.flatMap((definition) => callSites(definition.body)),
    )
  }
  const writers = new Set(STATUS_ARGUMENT)
  for (;;) {
    let grew = false
    for (const [name, calls] of sites) {
      if (writers.has(name)) continue
      if (calls.some((call) => writers.has(call.name))) {
        writers.add(name)
        grew = true
      }
    }
    if (!grew) break
  }

  const responders = new Map<string, Responder>()
  const delegates = new Map<string, string[]>()
  const callers = new Map<string, Set<string>>()
  for (const [name, calls] of sites) {
    const statuses = new Set<number>()
    const unread = new Set<string>()
    const followed = new Set<string>()
    for (const call of calls) {
      // Handing the raw response to something else — `ServeHTTP(c.Response(),
      // ...)` on /metrics — writes a status this reader never sees. Reading the
      // header off the context is not that: there the response object is the
      // receiver, not an argument.
      if (call.args.includes('c.Response()')) {
        unread.add(call.name)
        continue
      }
      if (!writers.has(call.name)) continue
      const constants = statusesIn(call.args)
      if (constants.length > 0) {
        for (const status of constants) statuses.add(status)
        continue
      }
      if (STATUS_ARGUMENT.has(call.name) || call.name === name || !definitions.has(call.name)) {
        if (STATUS_ARGUMENT.has(call.name)) unread.add(call.name)
        continue
      }
      // One name, two definitions: reading either would attribute a status to a
      // handler that may not have it, so neither is read.
      if ((definitions.get(call.name) ?? []).length > 1) unread.add(call.name)
      else if (isGuard(call.name, definitions)) followed.add(call.name)
      else unread.add(call.name)
    }
    responders.set(name, {
      path: (definitions.get(name) ?? [])[0]?.path ?? '',
      statuses,
      unread: [...unread],
      definitions: (definitions.get(name) ?? []).length,
    })
    delegates.set(name, [...followed])
    for (const callee of followed) callers.set(callee, (callers.get(callee) ?? new Set()).add(name))
  }

  // A guard's statuses and its own unread writers belong to everything that
  // stands behind it. Iterated to a fixed point so a chain of thin guards —
  // requireWorkflowAdmin to requireAdmin to WriteAdminAccessError — carries all
  // the way through, and so a cycle terminates.
  const work = [...responders.keys()]
  while (work.length > 0) {
    const name = work.pop() as string
    const responder = responders.get(name) as Responder
    const before = responder.statuses.size + responder.unread.length
    for (const callee of delegates.get(name) ?? []) {
      const target = responders.get(callee)
      if (!target) continue
      for (const status of target.statuses) responder.statuses.add(status)
      for (const writer of target.unread) {
        if (!responder.unread.includes(writer)) responder.unread.push(writer)
      }
    }
    if (responder.statuses.size + responder.unread.length !== before) {
      for (const caller of callers.get(name) ?? []) work.push(caller)
    }
  }
  return responders
}

/** `/things/:id` and `/things/{id}` are the same route written twice. */
function pathShape(key: string): string {
  return key.replace(/\{[^}]*\}/g, '{}')
}

const sortNumbers = (values: Iterable<number>) => [...values].sort((a, b) => a - b)

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
      // silent wrong answer this check exists to avoid.
      if (responder.definitions > 1) {
        unresolved.push({ operationId, reason: 'handler-ambiguous', detail: handlerName })
        continue
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

      if (responder.unread.length > 0) {
        unread.push({ operationId, writers: responder.unread })
        continue
      }

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
