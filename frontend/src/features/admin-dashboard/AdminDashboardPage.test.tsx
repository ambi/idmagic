import { IconUsers } from '@tabler/icons-react'
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminDashboardPage, DashboardMetricCard, SecurityTaskCard } from './AdminDashboardPage'
import { adminDashboardDictionary, friendlyEventName } from './AdminDashboardPage.i18n'

const baseProps = {
  actorUsername: 'taro',
  userCount: 10,
  activeUserCount: 8,
  disabledUserCount: 2,
  clientCount: 3,
  grantedConsentCount: 5,
}

describe('AdminDashboardPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouter(<AdminDashboardPage {...baseProps} />)
    expect(
      screen.getByRole('heading', { name: adminDashboardDictionary.en.title }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminDashboardDictionary.en.totalUsersLabel)).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(<AdminDashboardPage {...baseProps} />, {
      locale: 'ja',
    })
    expect(
      screen.getByRole('heading', { name: adminDashboardDictionary.ja.title }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminDashboardDictionary.ja.totalUsersLabel)).toBeInTheDocument()
  })

  it('does not show a recent-audit-events section (dashboard no longer displays audit events)', async () => {
    await renderWithRouter(<AdminDashboardPage {...baseProps} />)
    expect(screen.queryByText('Recent audit events')).not.toBeInTheDocument()
    expect(screen.queryByText('Audit events (24h)')).not.toBeInTheDocument()
  })

  it('localizes lifecycle-workflow audit event names without using the fallback formatter', () => {
    const expectedJapaneseNames = {
      LifecycleWorkflowCreated: 'ライフサイクルワークフローの作成',
      LifecycleWorkflowUpdated: 'ライフサイクルワークフローの更新',
      LifecycleWorkflowDeleted: 'ライフサイクルワークフローの削除',
      LifecycleWorkflowEnabled: 'ライフサイクルワークフローの有効化',
      LifecycleWorkflowDisabled: 'ライフサイクルワークフローの無効化',
      LifecycleWorkflowRunStarted: 'ライフサイクルワークフローの実行開始',
      LifecycleWorkflowRunSucceeded: 'ライフサイクルワークフローの実行成功',
      LifecycleWorkflowRunFailed: 'ライフサイクルワークフローの実行失敗',
      LifecycleWorkflowRunPartiallyFailed: 'ライフサイクルワークフローの実行一部失敗',
      LifecycleWorkflowRunCanceled: 'ライフサイクルワークフローの実行キャンセル',
      LifecycleWorkflowStepFailed: 'ライフサイクルワークフローのステップ失敗',
    }

    for (const [eventType, expectedName] of Object.entries(expectedJapaneseNames)) {
      expect(friendlyEventName(eventType, 'ja')).toBe(expectedName)
    }
  })
})

describe('dashboard presentation components', () => {
  it('renders metrics and navigation labels from props', () => {
    render(
      <ul>
        <DashboardMetricCard label="All Users" value={12} icon={IconUsers} tone="blue" />
        <SecurityTaskCard
          title="MFA"
          description="Enable MFA"
          href="/admin"
          actionLabel="Settings"
        />
      </ul>,
    )
    expect(screen.getByText('All Users')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Settings/ })).toHaveAttribute('href', '/admin')
  })
})
