import { IconUsers } from '@tabler/icons-react'
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DashboardMetricCard, DashboardQuickLink, SecurityTaskCard } from './AdminDashboardPage'

describe('dashboard presentation components', () => {
  it('renders metrics and navigation labels from props', () => {
    render(
      <ul>
        <DashboardMetricCard label="総ユーザー" value={12} icon={IconUsers} tone="blue" />
        <DashboardQuickLink
          href="/admin/users"
          icon={IconUsers}
          label="ユーザー"
          description="一覧"
        />
        <SecurityTaskCard title="MFA" description="有効にします" href="/admin" actionLabel="設定" />
      </ul>,
    )
    expect(screen.getByText('総ユーザー')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /ユーザー/ })).toHaveAttribute('href', '/admin/users')
    expect(screen.getByRole('link', { name: /設定/ })).toHaveAttribute('href', '/admin')
  })
})
