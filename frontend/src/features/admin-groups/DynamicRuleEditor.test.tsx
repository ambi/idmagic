import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { DynamicRuleEditor } from './DynamicRuleEditor'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import type { DynamicGroupRule } from '../../types'

const t = adminGroupsDictionary.en

const rule = (expression: string): DynamicGroupRule => ({
  group_id: 'group-1',
  tenant_id: 'tenant-1',
  expression,
  enabled: false,
  version: 1,
  referenced_attributes: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
})

describe('DynamicRuleEditor', () => {
  afterEach(() => vi.unstubAllGlobals())

  const stubUsers = () =>
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({ ok: true, status: 200, json: vi.fn().mockResolvedValue({ users: [] }) }),
      ),
    )

  // scenario: 新規ルールはビルダーで開く
  it('opens in builder mode when there is no saved expression', async () => {
    stubUsers()
    await renderWithRouter(
      <DynamicRuleEditor csrfToken="csrf" groupId="group-1" customAttributes={[]} />,
    )

    expect(screen.getByText(t.ruleBuilderHelp)).toBeInTheDocument()
    expect(screen.getByLabelText(t.ruleAttributeLabel)).toBeInTheDocument()
    expect(screen.queryByLabelText(t.dynamicRuleExpression)).not.toBeInTheDocument()
  })

  // scenario: T004 — ビルダーで表現できない保存済み式は詳細設定へフォールバック
  it('falls back to advanced mode for a saved OR expression', async () => {
    stubUsers()
    await renderWithRouter(
      <DynamicRuleEditor
        csrfToken="csrf"
        groupId="group-1"
        initialRule={rule('user.email_verified == true || user.department == "Sales"')}
        customAttributes={[]}
      />,
    )

    const textarea = screen.getByLabelText(t.dynamicRuleExpression) as HTMLTextAreaElement
    expect(textarea).toBeInTheDocument()
    expect(textarea.value).toBe('user.email_verified == true || user.department == "Sales"')
    expect(screen.getByText(t.ruleBuilderFallbackNotice)).toBeInTheDocument()
    expect(screen.queryByText(t.ruleBuilderHelp)).not.toBeInTheDocument()
  })
})
