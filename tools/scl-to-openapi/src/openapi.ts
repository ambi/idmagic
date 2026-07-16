/**
 * Pure transform: SCL `interfaces` (with http bindings) + `models` ->
 * OpenAPI 3.1.
 *
 * The second downstream artifact derived from the single SCL source. It reuses
 * the model->schema mapping from scl-to-jsonschema: every model becomes a
 * `components/schemas` entry, and each interface with an http binding becomes
 * an operation whose request/response bodies reference those schemas.
 *
 * Schemas are built with the `#/$defs/` ref base (the model generator's
 * convention) and rewritten to `#/components/schemas/` once at the end.
 */

import type {
  Binding,
  Field,
  Interface,
  SclBundle,
  SclDocument,
} from '../../scl-to-html/src/types.ts'
import {
  collectInterfaces,
  collectModels,
  fieldsToSchema,
  fieldToSchema,
  type JsonSchema,
  modelToSchema,
  rewriteRefs,
} from '../../scl-to-jsonschema/src/generate.ts'
import { buildAuthorizationMetadata } from './authorization.ts'

const DEFS = '#/$defs/'
const COMPONENTS = '#/components/schemas/'

type RequestStyle = 'query' | 'form' | 'xml' | 'json'

function firstLine(s: string | undefined): string | undefined {
  return s?.split('\n')[0]?.trim() || undefined
}

function pathParams(path: string): string[] {
  return [...path.matchAll(/\{([^}]+)\}/g)].map((m) => m[1] ?? '')
}

/** Decide how the request payload is carried, honouring both spec dialects. */
function requestStyle(binding: Binding, method: string): RequestStyle {
  const form = binding.request_form
  const body = binding.request_body
  if (form === 'query') return 'query'
  if (form === 'form') return 'form'
  if (body === 'xml') return 'xml'
  if (form === 'body' || body === 'json') return 'json'
  if (method === 'get' || method === 'delete' || method === 'head') return 'query'
  return 'json'
}

const MEDIA: Record<Exclude<RequestStyle, 'query'>, string> = {
  form: 'application/x-www-form-urlencoded',
  xml: 'application/xml',
  json: 'application/json',
}

function stringArray(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
}

function buildOperation(
  name: string,
  iface: Interface,
  binding: Binding,
  modelNames: ReadonlySet<string>,
): { method: string; operation: JsonSchema } {
  const method = String(binding.method ?? 'GET').toLowerCase()
  const path = String(binding.path ?? '')
  const operation: JsonSchema = { operationId: name }
  const summary = firstLine(iface.description)
  if (summary) operation.summary = summary
  if (iface.requires?.length) operation['x-scl-requires'] = [...iface.requires]
  if (iface.ensures?.length) operation['x-scl-ensures'] = [...iface.ensures]
  const contractDescription = [
    iface.requires?.length ? `Requires: ${iface.requires.join('; ')}` : '',
    iface.ensures?.length ? `Ensures: ${iface.ensures.join('; ')}` : '',
  ]
    .filter(Boolean)
    .join('\n\n')
  if (contractDescription) operation.description = contractDescription

  if (iface.access === 'public') {
    operation.security = []
  } else if (iface.access && typeof iface.access === 'object') {
    operation.security = [{ SclBearer: [] }]
    operation['x-scl-access'] = {
      policies: [...iface.access.policies].sort(),
      resource: { ...iface.access.resource },
    }
  }

  const parameters: JsonSchema[] = []
  const pathParameterNames = new Set(pathParams(path))
  for (const p of pathParameterNames) {
    parameters.push({ name: p, in: 'path', required: true, schema: { type: 'string' } })
  }

  const input: Record<string, Field> = iface.input ?? {}
  const style = requestStyle(binding, method)
  if (Object.keys(input).length > 0) {
    if (style === 'query') {
      for (const [fname, field] of Object.entries(input)) {
        if (pathParameterNames.has(fname)) continue
        parameters.push({
          name: fname,
          in: 'query',
          required: !field.optional,
          schema: fieldToSchema(field, modelNames),
        })
      }
    } else {
      operation.requestBody = {
        required: true,
        content: { [MEDIA[style]]: { schema: fieldsToSchema(input, modelNames) } },
      }
    }
  }
  if (parameters.length > 0) operation.parameters = parameters

  const responses: JsonSchema = {}
  const codes = stringArray(binding.successful_status_codes)
  const successCodes = codes.length > 0 ? codes : ['200']
  const output: Record<string, Field> = iface.output ?? {}
  const hasOutput = Object.keys(output).length > 0
  for (const code of successCodes) {
    const response: JsonSchema = { description: 'Success' }
    if (hasOutput && code.startsWith('2')) {
      response.content = { 'application/json': { schema: fieldsToSchema(output, modelNames) } }
    }
    responses[code] = response
  }
  const errors = stringArray(iface.errors)
  if (errors.length > 0) {
    responses.default = {
      description: `Errors: ${errors.join(', ')}`,
      content: {
        'application/json': {
          schema: { oneOf: errors.map((e) => ({ $ref: `${DEFS}${e}` })) },
        },
      },
    }
  }
  operation.responses = responses

  return { method, operation }
}

export function generateOpenApi(bundle: SclBundle | SclDocument): JsonSchema {
  const root = 'contexts' in bundle ? bundle.root : bundle
  const models = collectModels(bundle)
  const modelNames = new Set(Object.keys(models))

  const schemas: Record<string, JsonSchema> = {}
  for (const name of [...modelNames].sort()) {
    const model = models[name]
    if (model) schemas[name] = modelToSchema(model, modelNames)
  }

  const paths: Record<string, Record<string, JsonSchema>> = {}
  for (const [name, iface] of Object.entries(collectInterfaces(bundle))) {
    if (iface.access === 'internal') continue
    for (const binding of iface.bindings ?? []) {
      if (binding.kind !== 'http' || !binding.path) continue
      const path = String(binding.path)
      const { method, operation } = buildOperation(name, iface, binding, modelNames)
      let item = paths[path]
      if (!item) {
        item = {}
        paths[path] = item
      }
      if (item[method]) {
        const previous = String(item[method].operationId ?? 'unknown')
        throw new Error(
          `duplicate HTTP binding ${method.toUpperCase()} ${path}: ${previous}, ${name}`,
        )
      }
      item[method] = operation
    }
  }

  const authorization = buildAuthorizationMetadata(bundle)
  const components: JsonSchema = { schemas }
  if (authorization.groups.length > 0) {
    components.securitySchemes = {
      SclBearer: { type: 'http', scheme: 'bearer', bearerFormat: 'JWT' },
    }
  }
  const doc: JsonSchema = {
    openapi: '3.1.0',
    info: { title: `${root.system} API`, version: root.spec_version },
    paths,
    components,
    'x-scl-authorization': authorization,
  }
  // Schemas were built with the model generator's `#/$defs/` base; relocate
  // them under the OpenAPI components namespace.
  rewriteRefs(doc, DEFS, COMPONENTS)
  return doc
}
