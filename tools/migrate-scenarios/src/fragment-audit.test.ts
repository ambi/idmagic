import { describe, expect, it } from 'bun:test'
import { legacyErrorTypes, legacyFragments, missingFragments } from './fragment-audit.ts'

const legacy = `# Scenarios

### REQ-DEMO-001: 代行するエージェントは双方が関係を持つときだけ許可される
- ACTOR Agent
- GIVEN 代行されるユーザーが対象リソースに対して関係を持つ
- WHEN エージェントが CheckAccess を呼ぶ
  - ALT エージェント自身が同じ関係を持たない → 許可しない
  - ALT 提示トークンにスコープが含まれない → \`InvalidScopeError\` を返す
- THEN すべてを満たしたときにだけ許可する
`

const migrated = `# Feature: Demo

## Rule: REQ-DEMO-001 代行するエージェントは双方が関係を持つときだけ許可される

Primary actor: \`Agent\`

### Scenario Outline: 条件から許可を決める

- Given 代行されるユーザーが対象リソースに対して関係を <subject_relation>
- And エージェント自身が同じ関係を <actor_relation>
- And 提示トークンにスコープが <scope>
- When エージェントが CheckAccess を呼ぶ
- Then すべてを満たしたときにだけ <decision>

#### Examples: Decision table (Unique)

  | example_id | subject_relation | actor_relation | scope | decision |
  | --- | --- | --- | --- | --- |
  | EX-DEMO-001-01 | 持つ | 持つ | 含まれる | 許可する |
  | EX-DEMO-001-02 | 持つ | 持たない | 含まれる | 許可しない |
  | EX-DEMO-001-03 | 持つ | 持つ | 含まれない | \`InvalidScopeError\` を返す |
`

describe('scenario migration fragment audit', () => {
  it('splits an ALT into its condition and its outcome', () => {
    expect(legacyFragments(legacy)).toEqual([
      '代行されるユーザーが対象リソースに対して関係を持つ',
      'エージェントが CheckAccess を呼ぶ',
      'エージェント自身が同じ関係を持たない',
      '許可しない',
      '提示トークンにスコープが含まれない',
      '`InvalidScopeError` を返す',
      'すべてを満たしたときにだけ許可する',
    ])
  })

  it('reads the error types the legacy alternatives named', () => {
    expect([...legacyErrorTypes(legacy)]).toEqual(['InvalidScopeError'])
  })

  // 表の値を代入して初めて旧ステップの文になるので、生の本文だけを見ると
  // 正しく移した Decision Table まで欠落として報告してしまう。
  it('accepts a path a Decision table row carries only after expansion', () => {
    expect(missingFragments(legacy, migrated)).toEqual([])
  })

  it('reports a condition the migration dropped', () => {
    const withoutRow = migrated.replace(
      '  | EX-DEMO-001-02 | 持つ | 持たない | 含まれる | 許可しない |\n',
      '',
    )
    expect(missingFragments(legacy, withoutRow)).toEqual([
      'エージェント自身が同じ関係を持たない',
      '許可しない',
    ])
  })

  it('reports an outcome the migration dropped while keeping its condition', () => {
    const withoutOutcome = migrated.replace(
      '含まれない | `InvalidScopeError` を返す',
      '含まれない | 許可しない',
    )
    expect(missingFragments(legacy, withoutOutcome)).toEqual(['`InvalidScopeError` を返す'])
  })
})
