import { expect, test } from 'bun:test'
import {
  contractEvidenceInventory,
  contractEvidencePreflight,
} from './contract-evidence-inventory.ts'

const preflightDocument = {
  paths: {
    '/demo': { get: { operationId: 'ReadDemo' } },
    '/other': { get: { operationId: 'ReadOther' } },
    '/missing': { post: { operationId: 'Missing' } },
  },
}
const preflightFiles = [
  {
    path: 'backend/demo/handlers_http/routes.go',
    source: `package demo
import "example/backend/provider/ports"
func Register(g *echo.Group) { g.GET("/demo", handleReadDemo) }
func handleReadDemo(c *echo.Context) error { return c.JSON(http.StatusOK, value) }
`,
  },
  {
    path: 'backend/other/handlers_http/routes.go',
    source: `func Register(g *echo.Group) { g.GET("/other", handleReadOther) }
func handleReadOther(c *echo.Context) error { return c.JSON(http.StatusOK, value) }
`,
  },
  {
    path: 'backend/demo/handlers_http/read_test.go',
    source: `package demo
func TestRead(t *testing.T) { httptest.NewRequest("GET", "/demo", nil) }
`,
  },
]
const preflightVerification = ['ReadDemo', 'ReadOther', 'Missing'].map((operationId) => ({
  operationId,
  state: 'unverified' as const,
  reasons: ['variable response'],
}))

test('共有コードの影響する未解析操作を選び、候補だけでは参照不足を埋めない', () => {
  const result = contractEvidencePreflight(
    preflightDocument,
    preflightFiles,
    preflightVerification,
    'example',
    ['backend/shared/library/value.go'],
  )
  expect(result.missingReferences).toEqual(['ReadDemo', 'ReadOther', 'Missing'])
  expect(result.operations[0]).toMatchObject({
    operationId: 'ReadDemo',
    candidates: ['backend/demo/handlers_http/read_test.go'],
    references: [],
  })
  expect(result.execution).toBe('not-collected')
})

test('逆依存と不明な所有元を残し、無関係な操作とテストだけの変更を除く', () => {
  const selected = (paths: string[]) =>
    contractEvidencePreflight(
      preflightDocument,
      preflightFiles,
      preflightVerification,
      'example',
      paths,
    ).operations.map((operation) => operation.operationId)
  expect(selected(['backend/provider/domain/value.go'])).toEqual(['ReadDemo', 'Missing'])
  expect(selected(['backend/demo/handlers_http/read_test.go'])).toEqual([])
  expect(selected(['docs/development/testing.md'])).toEqual([])
  expect(selected(['spec/modules/demo/main.tsp'])).toEqual(['ReadDemo', 'ReadOther', 'Missing'])
})

test('受理した宣言を参照へ結び、静的に成功した操作を不足へ足さず、実行成功を推定しない', () => {
  const files = preflightFiles.map((file) =>
    file.path.endsWith('_test.go')
      ? {
          ...file,
          source: '//verify:operation ReadDemo: 本文と状態コード\nfunc TestRead(t *testing.T) {}\n',
        }
      : file,
  )
  const result = contractEvidencePreflight(
    preflightDocument,
    files,
    preflightVerification.map((operation) =>
      operation.operationId === 'ReadOther'
        ? { ...operation, state: 'verified' as const }
        : operation,
    ),
    'example',
    ['backend/shared/library/value.go'],
  )
  expect(result.missingReferences).toEqual(['Missing'])
  expect(result.operations[0]?.references).toEqual([
    { path: 'backend/demo/handlers_http/read_test.go', name: 'TestRead' },
  ])
  expect(result.operations.map((operation) => operation.operationId)).toEqual([
    'ReadDemo',
    'Missing',
  ])
  expect(result.execution).toBe('not-collected')
})

test('正規化で担当を推定できても全体検証の未解決経路を影響から外さない', () => {
  const document = { paths: { '/things/{id}': { get: { operationId: 'GetThing' } } } }
  const files = [
    {
      path: 'backend/demo/handlers_http/routes.go',
      source: 'g.GET("/things/:thing", handleGetThing)',
    },
    {
      path: 'backend/demo/handlers_http/get.go',
      source: 'func handleGetThing(c *echo.Context) error { return c.JSON(http.StatusOK, value) }',
    },
  ]
  const result = contractEvidencePreflight(
    document,
    files,
    [{ operationId: 'GetThing', state: 'unverified', reasons: ['variable response'] }],
    'example',
    ['backend/other/domain/value.go'],
  )
  expect(result.operations[0]?.handler?.path).toBe('backend/demo/handlers_http/get.go')
  expect(result.missingReferences).toEqual(['GetThing'])
})

test('未解析の操作を経路と本文検査の候補へ結び、候補を証拠に昇格させない', () => {
  const document = {
    paths: {
      '/things/{id}': { get: { operationId: 'GetThing' } },
      '/missing': { post: { operationId: 'Missing' } },
    },
  }
  const files = [
    {
      path: 'backend/demo/handlers_http/routes.go',
      source: 'g.GET("/things/:id", handleGetThing)',
    },
    {
      path: 'backend/demo/handlers_http/get.go',
      source: 'func handleGetThing(c *echo.Context) error { return c.JSON(http.StatusOK, value) }',
    },
    {
      path: 'backend/demo/handlers_http/get_test.go',
      source: 'httptest.NewRequest("GET", "/things/example", nil)',
    },
  ]
  expect(
    contractEvidenceInventory(document, files, [
      { operationId: 'GetThing', state: 'unverified', reasons: ['variable response'] },
      { operationId: 'Missing', state: 'unverified', reasons: ['route not found'] },
    ]),
  ).toEqual([
    {
      operationId: 'GetThing',
      method: 'GET',
      path: '/things/{id}',
      reasons: ['variable response'],
      handler: { name: 'handleGetThing', path: 'backend/demo/handlers_http/get.go' },
      candidates: ['backend/demo/handlers_http/get_test.go'],
    },
    {
      operationId: 'Missing',
      method: 'POST',
      path: '/missing',
      reasons: ['route not found'],
      handler: undefined,
      candidates: [],
    },
  ])
})
