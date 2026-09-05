import { describe, expect, it } from 'bun:test'
import { parseScenarioDocument } from './gherkin-scenarios.ts'

const valid = `# Feature: Demo

## Rule: REQ-DEMO-001 有効な要求を処理する

### Example: EX-DEMO-001-01 有効な要求

- Given 要求は有効である
- When 利用者が要求を送る
- Then 成功結果が返る

### Scenario Outline: 条件から結果を決める

- Given 条件は <condition> である
- When 利用者が要求を送る
- Then 結果は <outcome> である

#### Examples: Decision table (Unique)

  | example_id | condition | outcome |
  | --- | --- | --- |
  | EX-DEMO-001-02 | allowed | success |
  | EX-DEMO-001-03 | denied | refusal |
`

describe('Markdown with Gherkin scenarios', () => {
  it('parses rules, examples, and decision rows with the official parser', () => {
    const result = parseScenarioDocument(valid)

    expect(result.findings).toEqual([])
    expect(result.feature).toBe('Demo')
    expect(result.rules).toHaveLength(1)
    expect(result.rules[0]?.id).toBe('REQ-DEMO-001')
    expect(result.rules[0]?.examples.map((example) => example.id)).toEqual([
      'EX-DEMO-001-01',
      'EX-DEMO-001-02',
      'EX-DEMO-001-03',
    ])
    expect(result.rules[0]?.examples[2]?.decisionTable).toBe(true)
  })

  it('rejects examples outside a rule and a live rule without examples', () => {
    const source = `# Feature: Demo

### Example: EX-DEMO-001-01 orphan
- When 利用者が要求する
- Then 結果が返る

## Rule: REQ-DEMO-002 empty
`
    expect(parseScenarioDocument(source).findings.map((finding) => finding.message)).toEqual([
      'examples must belong to a Rule',
      'REQ-DEMO-002 must contain at least one example',
    ])
  })

  it('rejects an example id that does not belong to its parent rule', () => {
    expect(
      parseScenarioDocument(valid.replace('EX-DEMO-001-01', 'EX-DEMO-002-01')).findings,
    ).toEqual([
      {
        line: 5,
        message: 'EX-DEMO-002-01 must belong to REQ-DEMO-001',
      },
    ])
  })

  it('requires a unique example_id and nonempty outcome in every decision row', () => {
    const source = valid.replace('EX-DEMO-001-03 | denied | refusal', 'EX-DEMO-001-02 | allowed |')
    const messages = parseScenarioDocument(source).findings.map((finding) => finding.message)
    expect(messages).toContain('duplicate example id EX-DEMO-001-02')
    expect(messages).toContain('decision row EX-DEMO-001-02 must have a nonempty outcome')
    expect(messages).toContain('Decision table (Unique) has overlapping condition values')
  })

  it('treats any as a wildcard when checking Unique decision rows', () => {
    const source = valid.replace(
      'EX-DEMO-001-03 | denied | refusal',
      'EX-DEMO-001-03 | any | refusal',
    )
    expect(parseScenarioDocument(source).findings.map((finding) => finding.message)).toContain(
      'Decision table (Unique) has overlapping condition values',
    )
  })

  // 行に住所がなければテストは具体例を名指せないため、表そのものを拒否する。
  it('rejects an Examples table with no example_id column', () => {
    const source = valid.replace('| example_id | condition |', '| case | condition |')
    expect(parseScenarioDocument(source).findings.map((finding) => finding.message)).toEqual([
      'Examples table must contain an example_id column',
    ])
  })

  it('rejects an Examples row whose example_id is not an EX id', () => {
    const source = valid.replace('EX-DEMO-001-03 | denied', 'denied-row | denied')
    expect(parseScenarioDocument(source).findings.map((finding) => finding.message)).toEqual([
      'Examples row must contain an EX id in example_id',
    ])
  })

  it('does not infer example coverage from a parent REQ id', () => {
    const result = parseScenarioDocument(valid)
    const cited = new Set(['REQ-DEMO-001'])
    expect(result.rules[0]?.examples.every((example) => cited.has(example.id))).toBe(false)
  })
})
