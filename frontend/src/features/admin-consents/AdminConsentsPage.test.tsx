import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import {
  AdminConsentsPage,
  ConsentStateBadge,
  filterAdminConsents,
  NameWithId,
} from './AdminConsentsPage'
import { adminConsentsDictionary } from './AdminConsentsPage.i18n'
import type { AdminConsent } from '../../types'

const consent: AdminConsent = {
  user_id: 'user-1',
  preferred_username: 'taro',
  client_id: 'client-1',
  client_name: 'Payroll',
  scopes: ['openid', 'profile'],
  state: 'granted',
  granted_at: '2026-01-01T00:00:00Z',
  expires_at: '2026-02-01T00:00:00Z',
}

describe('filterAdminConsents', () => {
  it('matches a normalized query against user, client, state, and scopes', () => {
    expect(filterAdminConsents([consent], '  PROFILE ')).toEqual([consent])
    expect(filterAdminConsents([consent], 'missing')).toEqual([])
  })
})

describe('consent presentation components', () => {
  it('keeps an identifier as supporting text and renders the state', () => {
    render(
      <>
        <NameWithId name="taro" id="user-1" />
        <ConsentStateBadge state="granted" />
      </>,
    )

    expect(screen.getByText('taro')).toBeInTheDocument()
    expect(screen.getByText('user-1')).toBeInTheDocument()
    expect(screen.getByText(adminConsentsDictionary.ja.stateGranted)).toBeInTheDocument()
  })
})

describe('AdminConsentsPage', () => {
  it('renders in English by default', async () => {
    await renderWithRouterBase(<AdminConsentsPage csrfToken="csrf" consents={[]} />)
    expect(
      screen.getByRole('heading', { name: adminConsentsDictionary.en.pageTitle }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(adminConsentsDictionary.en.noMatchingConsentsNotice),
    ).toBeInTheDocument()
  })

  it('renders in Japanese when explicitly selected', async () => {
    await renderWithRouterBase(<AdminConsentsPage csrfToken="csrf" consents={[consent]} />, {
      locale: 'ja',
    })
    expect(
      screen.getByRole('heading', { name: adminConsentsDictionary.ja.pageTitle }),
    ).toBeInTheDocument()
    expect(screen.getByText(adminConsentsDictionary.ja.detailsHeading)).toBeInTheDocument()
  })
})
