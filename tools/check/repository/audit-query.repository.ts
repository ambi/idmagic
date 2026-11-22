import { expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const repositoryRoot = resolve(import.meta.dir, '../../..')

type Parameter = {
  name: string
  in: string
  required?: boolean
  style?: string
  explode?: boolean
  description?: string
  schema: { type: string; format?: string; items?: { type: string } }
}

type Operation = {
  operationId: string
  parameters: (Parameter | { $ref: string })[]
  requestBody?: unknown
}

//spec:covers REQ-AUDIT-001: 実装と同じ個別の条件と繰り返しの文字列フィルターを生成クライアントへ公開する。
it('監査の6操作は個別のクエリと繰り返しの文字列フィルターを宣言する', () => {
  const document = JSON.parse(
    readFileSync(resolve(repositoryRoot, 'spec/generated/openapi/idmagic.openapi.json'), 'utf8'),
  ) as {
    paths: Record<string, Record<string, Operation>>
    components: { parameters: Record<string, Parameter> }
  }
  const routes = [
    ['/api/admin/v1/audit-events', 'get', 'ListAdminAuditEvents'],
    ['/api/admin/v1/system/audit-events', 'get', 'ListSystemAuditEvents'],
    ['/api/admin/v1/audit-events/export', 'get', 'ExportAdminAuditEvents'],
    ['/api/admin/v1/system/audit-events/export', 'get', 'ExportSystemAuditEvents'],
    ['/api/admin/v1/audit-events/exports', 'post', 'StartAdminAuditExport'],
    ['/api/admin/v1/system/audit-events/exports', 'post', 'StartSystemAuditExport'],
  ] as const

  for (const [path, method, operationId] of routes) {
    const operation = document.paths[path]![method]!
    expect(operation.operationId).toBe(operationId)
    expect(operation.requestBody).toBeUndefined()
    const parameters = operation.parameters
      .map((parameter) =>
        '$ref' in parameter
          ? document.components.parameters[parameter.$ref.replace('#/components/parameters/', '')]!
          : parameter,
      )
      .filter((parameter) => parameter.in === 'query')
    expect(parameters.map((parameter) => parameter.name).sort()).toEqual([
      'after',
      'before',
      'category',
      'cursor',
      'filter',
      'limit',
      'q',
      'type',
      'user_id',
      'username',
    ])
    for (const parameter of parameters) {
      expect(parameter.required).not.toBe(true)
      if (parameter.name === 'filter') {
        expect(parameter.schema).toEqual({ type: 'array', items: { type: 'string' } })
        // OpenAPI の form / explode=true は同名のパラメーターを繰り返す。
        expect(parameter.style ?? 'form').toBe('form')
        expect(parameter.explode ?? true).toBe(true)
        expect(parameter.description).toContain('field:operator:value[,value2]')
      } else if (parameter.name === 'limit') {
        expect(parameter.schema.type).toBe('integer')
        expect(parameter.schema.format).toBe('int64')
      } else {
        expect(parameter.schema.type).toBe('string')
        if (parameter.name === 'after' || parameter.name === 'before') {
          expect(parameter.schema.format).toBe('date-time')
        }
      }
    }
  }
})
