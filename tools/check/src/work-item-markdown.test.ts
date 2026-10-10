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
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: The parser fixture has no release reader.
  references: []
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

  it('docs/formats/work-item-format.md が示す日本語見出しを同じ項目へ解決する', () => {
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

  // 英語の表記を足し忘れた項目は、その行が黙って読み飛ばされ、必須項目の欠落として
  // 報告される。すべての項目を一度に並べ、どの対応が欠けても値の不一致として現れるようにする。
  it('docs/formats/work-item-format.md が示す日本語のフィールド名を英語と同じ項目へ解決する', () => {
    const source = `---
status: completed
---

# 日本語のフィールド名の完了記録

## 完了

- **完了日**: 2026-10-07
- **要約**: 日本語のフィールド名で完了を記録した。
- **RED の証拠**:
  - **テスト**: 検査 A
  - **要件**: N/A: 開発ツール
  - **観測した失敗**: 検査 A が失敗した。
  - **検出できる理由**: 項目が欠けると検査 A が落ちる。
- **受け入れ RED の証拠**:
  - **テスト**: 検査 B
  - **要件**: REQ-SYSTEM-001
  - **観測した失敗**: 検査 B が失敗した。
  - **検出できる理由**: 結果を表明する。
- **単体 RED の証拠**:
  - **テスト**: 検査 C
  - **要件**: REQ-SYSTEM-002
  - **観測した失敗**: 検査 C が失敗した。
  - **検出できる理由**: 判断を表明する。
- **主要ユースケースの証拠**:
  - id: start-task
    red: 開始しなかった。
    fault_injection: 発行を削除すると失敗した。
- **独立した検証**: 別のエージェントが確認した。
- **変更耐性の結果**: 変異を検出した。
- **検証結果**:
  - mise run test-tools - 成功
`

    expect(parseFrontmatterAndMarkdown('wi-999-japanese-fields.md', source).completion).toEqual({
      completed_at: '2026-10-07',
      summary: '日本語のフィールド名で完了を記録した。',
      red_evidence: {
        test: '検査 A',
        requirement: 'N/A: 開発ツール',
        observed_failure: '検査 A が失敗した。',
        detection_reason: '項目が欠けると検査 A が落ちる。',
      },
      acceptance_red_evidence: {
        test: '検査 B',
        requirement: 'REQ-SYSTEM-001',
        observed_failure: '検査 B が失敗した。',
        detection_reason: '結果を表明する。',
      },
      unit_red_evidence: {
        test: '検査 C',
        requirement: 'REQ-SYSTEM-002',
        observed_failure: '検査 C が失敗した。',
        detection_reason: '判断を表明する。',
      },
      primary_use_case_evidence: [
        {
          id: 'start-task',
          red: '開始しなかった。',
          fault_injection: '発行を削除すると失敗した。',
        },
      ],
      independent_verification: '別のエージェントが確認した。',
      change_resistance: '変異を検出した。',
      verification: ['mise run test-tools - 成功'],
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
