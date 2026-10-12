import { expect, test } from 'bun:test'
import { contractEvidenceInventory } from './contract-evidence-inventory.ts'

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
