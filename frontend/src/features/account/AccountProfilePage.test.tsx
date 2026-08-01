import { afterEach, describe, it, expect, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import {
  AccountProfileEditPage,
  AccountProfilePresentation,
  draftFromProfile,
  textToValue,
  valueToText,
} from './AccountProfilePage'
import { EditableAttributeGroups } from './AccountProfileAttributes'
import type { AccountProfile, AttributeValue, UserAttributeDef } from '../../types'

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const stringDef: UserAttributeDef = {
  key: 'nickname',
  type: 'string',
  multi_valued: false,
  required: false,
  editable_by_user: true,
  visibility: 'self_readable',
  pii: false,
}

describe('valueToText', () => {
  it('converts a string value', () => {
    expect(valueToText({ type: 'string', string: 'hello' })).toBe('hello')
  })

  it('converts a number value', () => {
    expect(valueToText({ type: 'number', number: 42 })).toBe('42')
  })

  it('converts a boolean value', () => {
    expect(valueToText({ type: 'boolean', boolean: true })).toBe('true')
    expect(valueToText({ type: 'boolean', boolean: false })).toBe('false')
  })

  it('joins a string_array value with commas', () => {
    expect(valueToText({ type: 'string_array', string_array: ['a', 'b'] })).toBe('a, b')
  })
})

describe('textToValue', () => {
  it('returns undefined for empty string input', () => {
    expect(textToValue(stringDef, '  ')).toBeUndefined()
  })

  it('trims and wraps string input', () => {
    expect(textToValue(stringDef, '  hello  ')).toEqual({ type: 'string', string: 'hello' })
  })

  it('parses number input', () => {
    expect(textToValue({ ...stringDef, type: 'number' }, '42')).toEqual({
      type: 'number',
      number: 42,
    })
  })

  it('parses boolean input regardless of emptiness', () => {
    expect(textToValue({ ...stringDef, type: 'boolean' }, 'true')).toEqual({
      type: 'boolean',
      boolean: true,
    })
  })

  it('splits string_array input on commas and trims each item', () => {
    expect(textToValue({ ...stringDef, type: 'string_array' }, 'a, b ,  c')).toEqual({
      type: 'string_array',
      string_array: ['a', 'b', 'c'],
    })
  })

  it('returns undefined for an empty string_array input', () => {
    expect(textToValue({ ...stringDef, type: 'string_array' }, '  ')).toBeUndefined()
  })
})

describe('draftFromProfile', () => {
  it('builds a text draft from editable attribute values', () => {
    const value: AttributeValue = { type: 'string', string: 'Taro' }
    const profile: Partial<AccountProfile> = {
      editable_attributes: [stringDef],
      attributes: { nickname: value },
    }
    expect(draftFromProfile(profile as AccountProfile)).toEqual({ nickname: 'Taro' })
  })

  it('uses an empty string for attributes without a value', () => {
    const profile: Partial<AccountProfile> = {
      editable_attributes: [stringDef],
      attributes: {},
    }
    expect(draftFromProfile(profile as AccountProfile)).toEqual({ nickname: '' })
  })
})

describe('AccountProfilePresentation', () => {
  const profile: AccountProfile = {
    sub: 'user-1',
    preferred_username: 'taro',
    name: 'Taro Yamada',
    email: 'taro@example.com',
    email_verified: true,
    mfa_enrolled: false,
    status: 'active',
    attributes: {},
    readable_attributes: [],
    editable_attributes: [],
  }

  it('renders profile fields', async () => {
    await renderWithRouter(
      <AccountProfilePresentation
        profile={profile}
        isAdmin={false}
        notice=""
        onDismissNotice={mock()}
      />,
    )
    expect(screen.getByText('Taro Yamada')).toBeInTheDocument()
    expect(screen.getByText('確認済み')).toBeInTheDocument()
  })

  it('renders readable custom attributes', async () => {
    await renderWithRouter(
      <AccountProfilePresentation
        profile={{
          ...profile,
          attributes: { nickname: { type: 'string', string: 'たろう' } },
          readable_attributes: [stringDef],
        }}
        isAdmin={false}
        notice=""
        onDismissNotice={mock()}
      />,
    )
    expect(screen.getByText('nickname')).toBeInTheDocument()
    expect(screen.getByText('たろう')).toBeInTheDocument()
  })

  it('shows a notice toast when provided', async () => {
    await renderWithRouter(
      <AccountProfilePresentation
        profile={profile}
        isAdmin={false}
        notice="プロフィールを更新しました。"
        onDismissNotice={mock()}
      />,
    )
    expect(screen.getByText('プロフィールを更新しました。')).toBeInTheDocument()
  })

  it('calls onDismissNotice when the toast is dismissed', async () => {
    const onDismissNotice = mock()
    await renderWithRouter(
      <AccountProfilePresentation
        profile={profile}
        isAdmin={false}
        notice="プロフィールを更新しました。"
        onDismissNotice={onDismissNotice}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /閉じる|dismiss/i }))
    expect(onDismissNotice).toHaveBeenCalledTimes(1)
  })
})

describe('EditableAttributeGroups', () => {
  it('reports edited custom attribute values', async () => {
    const onChange = mock()
    await renderWithRouter(
      <EditableAttributeGroups
        defs={[stringDef]}
        values={{ nickname: 'before' }}
        onChange={onChange}
      />,
    )

    fireEvent.change(screen.getByLabelText('nickname'), { target: { value: 'after' } })
    expect(onChange).toHaveBeenCalledWith('nickname', 'after')
  })
})

describe('AccountProfileEditPage', () => {
  const originalLocation = window.location
  const profile: AccountProfile = {
    sub: 'user-1',
    preferred_username: 'taro',
    name: 'Taro Yamada',
    email: 'taro@example.com',
    email_verified: true,
    mfa_enrolled: false,
    status: 'active',
    attributes: {},
    readable_attributes: [],
    editable_attributes: [],
  }

  afterEach(() => restoreGlobals())

  it('saves the profile and redirects with a success notice', async () => {
    stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/acme/account/profile/edit',
      assign: mock(),
    })
    stubGlobal('fetch', mock().mockResolvedValue(response(200, profile)))
    await renderWithRouter(
      <AccountProfileEditPage csrfToken="csrf" profile={profile} isAdmin={false} />,
    )
    fireEvent.change(screen.getByLabelText('表示名'), { target: { value: 'Jiro Yamada' } })
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    await waitFor(() =>
      expect(window.location.assign).toHaveBeenCalledWith(
        '/realms/acme/account/profile?notice=success',
      ),
    )
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/profile'),
      expect.objectContaining({ body: expect.stringContaining('Jiro Yamada') }),
    )
  })

  it('shows an error and keeps the form when saving fails', async () => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(400, { message: '表示名を保存できませんでした。' })),
    )
    await renderWithRouter(
      <AccountProfileEditPage csrfToken="csrf" profile={profile} isAdmin={false} />,
    )
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    expect(await screen.findByText('表示名を保存できませんでした。')).toBeInTheDocument()
    expect(window.location.assign).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: '保存' })).toBeEnabled()
  })
})
