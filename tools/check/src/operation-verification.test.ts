import { expect, test } from 'bun:test'
import { operationVerification } from './operation-verification.ts'

test('未解決理由が二つあっても未検証の操作を一度だけ数える', () => {
  const result = operationVerification(
    ['Create', 'Read', 'Delete'],
    [{ operationId: 'Delete', message: 'wrong status' }],
    [
      { operationId: 'Create', reason: 'request unread' },
      { operationId: 'Create', reason: 'response unread' },
      { operationId: 'Delete', reason: 'dynamic response' },
    ],
  )
  expect(result).toEqual([
    { operationId: 'Create', state: 'unverified', reasons: ['request unread', 'response unread'] },
    { operationId: 'Read', state: 'verified', reasons: [] },
    { operationId: 'Delete', state: 'failed', reasons: ['wrong status', 'dynamic response'] },
  ])
})
