import { describe, it, expect, mock } from 'bun:test'
import { render, screen, fireEvent } from '@testing-library/react'
import {
  ActivityHistorySection,
  SessionsSection,
  accountActivityMethodSummary,
  formatAccountActivityDateTime,
} from './AccountActivityPage'
import type { AccountSession, AccountSignInActivity } from '../../types'
import { accountActivityDictionary } from './AccountActivityPage.i18n'

const t = accountActivityDictionary.en

describe('formatAccountActivityDateTime', () => {
  it('formats a valid ISO date string', () => {
    expect(formatAccountActivityDateTime('2026-01-15T10:30:00Z')).toContain('2026')
  })
})

describe('accountActivityMethodSummary', () => {
  it('returns the English fallback for an empty amr list', () => {
    expect(accountActivityMethodSummary([])).toBe(t.unknownMethod)
  })

  it('joins known amr codes with a plus sign', () => {
    expect(accountActivityMethodSummary(['pwd', 'otp'])).toBe(`${t.pwd} + ${t.otp}`)
  })

  it('falls back to the raw code for unknown amr values', () => {
    expect(accountActivityMethodSummary(['unknown-code'])).toBe('unknown-code')
  })
})

describe('SessionsSection', () => {
  const session: AccountSession = {
    id: 'session-1',
    current: false,
    amr: ['pwd'],
    acr: '1',
    started_at: '2026-01-01T00:00:00Z',
    expires_at: '2026-01-02T00:00:00Z',
  }

  const baseProps = {
    sessions: [session],
    busyId: null,
    busyOthers: false,
    onRevoke: mock(),
    onRevokeOthers: mock(),
  }

  it('shows an empty state when there are no sessions', () => {
    render(<SessionsSection {...baseProps} sessions={[]} />)
    expect(screen.getByText(t.noSessions)).toBeInTheDocument()
  })

  it('renders sessions with a revoke button for non-current sessions', () => {
    render(<SessionsSection {...baseProps} />)
    expect(screen.getByRole('button', { name: t.end })).toBeInTheDocument()
  })

  it('calls onRevoke with the session id when End is clicked', () => {
    const onRevoke = mock()
    render(<SessionsSection {...baseProps} onRevoke={onRevoke} />)
    fireEvent.click(screen.getByRole('button', { name: t.end }))
    expect(onRevoke).toHaveBeenCalledWith('session-1')
  })

  it('shows End other sessions only when other sessions exist', () => {
    render(<SessionsSection {...baseProps} sessions={[{ ...session, current: true }]} />)
    expect(screen.queryByRole('button', { name: t.endOther })).not.toBeInTheDocument()
  })

  it('calls onRevokeOthers when the button is clicked', () => {
    const onRevokeOthers = mock()
    render(<SessionsSection {...baseProps} onRevokeOthers={onRevokeOthers} />)
    fireEvent.click(screen.getByRole('button', { name: t.endOther }))
    expect(onRevokeOthers).toHaveBeenCalledTimes(1)
  })
})

describe('ActivityHistorySection', () => {
  const activity: AccountSignInActivity = {
    occurred_at: '2026-01-01T00:00:00Z',
    amr: ['pwd', 'otp'],
  }

  it('shows an empty state when there is no sign-in history', () => {
    render(<ActivityHistorySection activities={[]} />)
    expect(screen.getByText(t.noHistory)).toBeInTheDocument()
  })

  it('renders activity rows', () => {
    render(<ActivityHistorySection activities={[activity]} />)
    expect(screen.getByText(`${t.pwd} + ${t.otp}`)).toBeInTheDocument()
  })
})
