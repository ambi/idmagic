import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { OnDemandAndResyncPanel } from './AdminApplicationProvisioningOnDemand'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import type { AdminGroup, AdminUser } from '../../types'

const t = provisioningDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const user: AdminUser = {
  id: 'user-1',
  preferred_username: 'taro',
  name: 'Taro Yamada',
  email: 'taro@example.com',
  email_verified: true,
  mfa_enrolled: false,
  roles: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const group: AdminGroup = {
  id: 'group-1',
  tenant_id: 'tenant-1',
  name: 'Engineering',
  roles: [],
  member_count: 1,
  created_at: '2026-01-01T00:00:00Z',
}

function stubFetch(handler: (url: string, init?: RequestInit) => ReturnType<typeof response>) {
  stubGlobal(
    'fetch',
    mock((url: string, init?: RequestInit) => Promise.resolve(handler(url, init))),
  )
}

// chooseOption は Select (Base UI Select ベース) をクリックで開き、指定ラベルの
// option を選ぶ (AdminApplicationAssignments.test.tsx と同じパターン)。
async function chooseOption(triggerName: string | RegExp, optionName: string) {
  fireEvent.click(screen.getByRole('combobox', { name: triggerName }))
  const option = await screen.findByRole('option', { name: optionName })
  fireEvent.pointerDown(option)
  fireEvent.click(option)
}

// chooseComboboxOption は検索可能な SearchableSelect (Base UI combobox ベース) を対象に、
// 入力欄への mousedown でポップアップを開き (openOnInputClick は mousedown-only で判定される)、
// 指定ラベルの option を選ぶ (AdminGroupEditPage.test.tsx と同じパターン)。
async function chooseComboboxOption(inputName: string | RegExp, optionName: string) {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: inputName }))
  const option = await screen.findByRole('option', { name: optionName })
  fireEvent.click(option)
}

describe('OnDemandAndResyncPanel', () => {
  afterEach(() => restoreGlobals())

  it('lets an admin pick a user from a list instead of typing a raw subject ID', async () => {
    stubFetch((url) => {
      if (url.includes('/api/admin/v1/users')) return response(200, { users: [user] })
      if (url.includes('/api/admin/v1/groups')) return response(200, { groups: [] })
      return response(200, {})
    })
    await renderWithRouter(<OnDemandAndResyncPanel csrfToken="csrf" applicationID="app-1" />)

    await chooseComboboxOption(t.onDemandSelectUserPlaceholder, user.preferred_username)
    fireEvent.click(screen.getByRole('button', { name: t.onDemandButton }))

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/provisioning/on-demand'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ subject_type: 'user', subject_id: user.id }),
        }),
      ),
    )
  })

  it('switches to the group list and resets the selection when the subject type changes', async () => {
    stubFetch((url) => {
      if (url.includes('/api/admin/v1/users')) return response(200, { users: [user] })
      if (url.includes('/api/admin/v1/groups')) return response(200, { groups: [group] })
      return response(200, {})
    })
    await renderWithRouter(<OnDemandAndResyncPanel csrfToken="csrf" applicationID="app-1" />)

    await chooseComboboxOption(t.onDemandSelectUserPlaceholder, user.preferred_username)
    await chooseOption(t.onDemandSubjectTypeUser, t.onDemandSubjectTypeGroup)

    expect(
      await screen.findByRole('combobox', { name: t.onDemandSelectGroupPlaceholder }),
    ).toHaveValue('')
    expect(screen.queryByText(user.preferred_username)).not.toBeInTheDocument()
  })

  it('shows an error when the user/group lists fail to load', async () => {
    stubGlobal(
      'fetch',
      mock(() => Promise.reject(new Error('network down'))),
    )
    await renderWithRouter(<OnDemandAndResyncPanel csrfToken="csrf" applicationID="app-1" />)

    expect(await screen.findByText(t.onDemandSubjectsLoadFailedError)).toBeInTheDocument()
  })
})
