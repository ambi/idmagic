import { expect, test } from 'bun:test'
import { renderVerificationReport } from './report.ts'

test('参照数と実行結果を区別し、別の作業ツリーの結果を stale と表示する', () => {
  const text = renderVerificationReport(
    {
      input: { revision: 'old', fingerprint: 'old-tree' },
      tests: [],
      references: [
        {
          package: 'example',
          name: 'TestRule',
          path: 'test.go',
          normativeIds: ['REQ-DEMO-001'],
          operationIds: [],
          state: 'not-run',
        },
      ],
      databases: [],
      checks: [],
      failures: [],
      changeResistance: { state: 'not-collected' },
    },
    'new-tree',
  )
  expect(text).toContain('stale; unverified for the current tree')
  expect(text).toContain('| not-run | 0 | 1 |')
  expect(text).toContain('not-collected')
})
