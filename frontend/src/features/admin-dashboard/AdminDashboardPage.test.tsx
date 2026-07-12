import { IconUsers } from '@tabler/icons-react'
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminAuditEvent } from '../../types'
import {
  AdminDashboardPage,
  DashboardMetricCard,
  DashboardQuickLink,
  SecurityTaskCard,
} from './AdminDashboardPage'
import { adminDashboardDictionary, friendlyEventName } from './AdminDashboardPage.i18n'

const recentEvents: AdminAuditEvent[] = [
  {
    id: 'evt-1',
    tenant_id: 'acme',
    type: 'UserCreated',
    occurred_at: '2026-01-15T10:30:00Z',
    payload: { sub: 'user-1' },
  } as AdminAuditEvent,
]

const baseProps = {
  actorUsername: 'taro',
  actorRoles: ['admin'],
  userCount: 10,
  activeUserCount: 8,
  disabledUserCount: 2,
  clientCount: 3,
  grantedConsentCount: 5,
  auditEventCount24h: 12,
  recentEvents: [],
}

describe('AdminDashboardPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouter(<AdminDashboardPage {...baseProps} />)
    expect(
      screen.getByRole('heading', { name: adminDashboardDictionary.en.title }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminDashboardDictionary.en.totalUsersLabel)).toBeInTheDocument()
    expect(screen.getByText(adminDashboardDictionary.en.emptyRecentEvents)).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouter(<AdminDashboardPage {...baseProps} recentEvents={recentEvents} />, {
      locale: 'ja',
    })
    expect(
      screen.getByRole('heading', { name: adminDashboardDictionary.ja.title }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminDashboardDictionary.ja.totalUsersLabel)).toBeInTheDocument()
    expect(screen.getByText(friendlyEventName('UserCreated', 'ja'))).toBeInTheDocument()
  })
})

describe('dashboard presentation components', () => {
  it('renders metrics and navigation labels from props', () => {
    render(
      <ul>
        <DashboardMetricCard label="All Users" value={12} icon={IconUsers} tone="blue" />
        <DashboardQuickLink
          href="/admin/users"
          icon={IconUsers}
          label="Users"
          description="List users"
        />
        <SecurityTaskCard
          title="MFA"
          description="Enable MFA"
          href="/admin"
          actionLabel="Settings"
        />
      </ul>,
    )
    expect(screen.getByText('All Users')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Users/ })).toHaveAttribute('href', '/admin/users')
    expect(screen.getByRole('link', { name: /Settings/ })).toHaveAttribute('href', '/admin')
  })
})
