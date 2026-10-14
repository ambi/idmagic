import { describe, expect, it } from 'bun:test'
import { parseFrontmatterAndMarkdown, validateMarkdownRecord } from './work-item-markdown.ts'

describe('work-item Markdown sections', () => {
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
})
