import { afterEach, describe, expect, it } from 'bun:test'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { parseFrontmatterAndMarkdown, validateMarkdownRecord } from './work-item-markdown.ts'

const temporaryDirectories: string[] = []

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })))
})

describe('work-item Markdown completion evidence', () => {
  it('parses structured RED and stronger completion evidence', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'idmagic-work-item-'))
    temporaryDirectories.push(directory)
    const path = join(directory, 'wi-410-parser-evidence.md')
    await writeFile(
      path,
      `---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
evidence_policy: risk-based-v1
---

# Parse completion evidence

## Motivation

Keep evidence machine-readable.

## Scope

- Markdown parser

## Out of Scope

- Product behavior

## Verification

- mise run test-tools

## Risk Notes

The parser must reject incomplete evidence.

## Completion

- **Completed At**: 2026-08-23
- **Summary**: Parsed the evidence.
- **RED Evidence**:
  - **Test**: parser rejects missing evidence
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete record failed validation
  - **Detection Reason**: each nested field maps to a required schema property
- **Independent Verification**: reviewed by another agent
- **Change-Resistance Results**: removing one field fails validation
- **Verification Results**:
  - mise run test-tools - passed
`,
    )

    expect(validateMarkdownRecord(path, await Bun.file(path).text(), 'work-item').findings).toEqual(
      [],
    )
  })

  it('parses separate Acceptance RED and Unit RED evidence', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'idmagic-work-item-'))
    temporaryDirectories.push(directory)
    const path = join(directory, 'wi-412-parser-evidence.md')
    await writeFile(
      path,
      `---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
evidence_policy: risk-based-v2
---

# Parse separate RED evidence

## Motivation

Keep both evidence boundaries machine-readable.

## Scope

- Markdown parser

## Out of Scope

- Product behavior

## Verification

- mise run test-tools

## Risk Notes

The parser must reject either missing boundary.

## Completion

- **Completed At**: 2026-08-23
- **Summary**: Parsed both evidence boundaries.
- **Acceptance RED Evidence**:
  - **Test**: parser rejects missing acceptance evidence
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete record failed validation
  - **Detection Reason**: the acceptance field is independently required
- **Unit RED Evidence**:
  - **Test**: parser rejects missing unit evidence
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete record failed validation
  - **Detection Reason**: the unit field is independently required
- **Independent Verification**: reviewed by another agent
- **Change-Resistance Results**: removing either field fails validation
- **Verification Results**:
  - mise run test-tools - passed
`,
    )

    expect(validateMarkdownRecord(path, await Bun.file(path).text(), 'work-item').findings).toEqual(
      [],
    )
  })

  it('WORK_ITEM_FORMAT.md が示す日本語見出しを同じ項目へ解決する', () => {
    const source = `---
status: pending
authors: [tn]
risk: low
created_at: 2026-09-19
priority: p3
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "検査器の見出し解釈だけを変える。"
---

# 日本語見出しの作業項目

## 動機

フォーマット文書が示す見出しで書いた記録も解析できる必要がある。

## 対象範囲

- Markdown の解析

## 対象外

- プロダクトの振る舞い

## 設計

節の名前だけを対応表で解決する。

## 計画

1. 対応表を加える。

## タスク

- [ ] T001 [App] 対応表を加える。

## 検証

- mise run test-tools

## リスク

見出しの取り違えで必須項目が欠落として報告される。
`

    const data = parseFrontmatterAndMarkdown('wi-999-japanese-headings.md', source)

    expect(data.title).toBe('日本語見出しの作業項目')
    expect(data.motivation).toContain('フォーマット文書')
    expect(data.scope).toContain('Markdown の解析')
    expect(data.out_of_scope).toEqual(['プロダクトの振る舞い'])
    expect(data.plan).toContain('対応表を加える')
    expect(data.tasks).toContain('T001')
    expect(data.verification).toEqual(['mise run test-tools'])
    expect(data.risk_notes).toContain('見出しの取り違え')
    expect(
      validateMarkdownRecord('wi-999-japanese-headings.md', source, 'work-item').findings,
    ).toEqual([])
  })

  it('日本語見出しの「完了」を完了記録として解析する', () => {
    const source = `---
status: completed
---

# 日本語見出しの完了記録

## 完了

- **Completed At**: 2026-09-19
- **Summary**: 日本語見出しで完了を記録した。
`

    expect(
      parseFrontmatterAndMarkdown('wi-999-japanese-completion.md', source).completion,
    ).toMatchObject({
      completed_at: '2026-09-19',
      summary: '日本語見出しで完了を記録した。',
    })
  })

  it('parses primary-use-case completion evidence as structured YAML', () => {
    const source = `---
status: completed
---

# Primary use-case evidence

## Completion

- **Completed At**: 2026-08-30
- **Summary**: Recorded primary use-case evidence.
- **Primary Use Case Evidence**:
  - id: configured-delivery
    unit_red: the use-case test observed no outgoing effect
    e2e_red: the external entry produced no delivery
    unit_fault_injection: removing the branch made the unit test fail
    e2e_fault_injection: disconnecting the adapter made the E2E test fail
`

    expect(parseFrontmatterAndMarkdown('wi-445-demo.md', source).completion).toMatchObject({
      primary_use_case_evidence: [
        {
          id: 'configured-delivery',
          unit_red: 'the use-case test observed no outgoing effect',
          e2e_red: 'the external entry produced no delivery',
          unit_fault_injection: 'removing the branch made the unit test fail',
          e2e_fault_injection: 'disconnecting the adapter made the E2E test fail',
        },
      ],
    })
  })
})
