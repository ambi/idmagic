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
  expect(text).toContain('Result cache: not recorded')
})

test('キャッシュから再生したパッケージを、今回実行したパッケージと区別して表示する', () => {
  const report = {
    input: { revision: 'head', fingerprint: 'tree' },
    tests: [],
    references: [],
    databases: [],
    checks: [],
    failures: [],
    changeResistance: { state: 'not-collected' },
  }
  expect(
    renderVerificationReport({ ...report, cachedPackages: ['example/a', 'example/b'] }, 'tree'),
  ).toContain(
    'Result cache: **2 package(s) replayed from an earlier run with the same inputs** (example/a, example/b)',
  )
  expect(renderVerificationReport({ ...report, cachedPackages: [] }, 'tree')).toContain(
    'Result cache: every package executed in this run',
  )
})
