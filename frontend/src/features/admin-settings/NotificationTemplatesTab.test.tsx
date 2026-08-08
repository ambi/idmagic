import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { LocaleProvider } from '../../lib/i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { NotificationTemplatesTab } from './NotificationTemplatesTab'
import { notificationTemplatesTabDictionary } from './NotificationTemplatesTab.i18n'

const t = notificationTemplatesTabDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const listBody = {
  supported_locales: ['ja', 'en'],
  templates: [
    {
      template_key: 'password_reset',
      locale: 'ja',
      customized: false,
      subject: '{{product_name}} のパスワード再設定',
    },
    {
      template_key: 'password_reset',
      locale: 'en',
      customized: true,
      subject: 'Reset your password',
      updated_at: '2026-07-25T00:00:00Z',
    },
  ],
}

const detailBody = {
  template_key: 'password_reset',
  locale: 'ja',
  customized: false,
  subject: '{{product_name}} のパスワード再設定',
  body_text: '{{user_display_name}} さん {{reset_url}}',
  body_html: '<p>{{user_display_name}} さん</p>',
  default_subject: '{{product_name}} のパスワード再設定',
  default_body_text: '{{user_display_name}} さん {{reset_url}}',
  default_body_html: '<p>{{user_display_name}} さん</p>',
  placeholders: ['product_name', 'user_display_name', 'reset_url', 'expires_in_minutes'],
}

function renderTab() {
  return render(
    <LocaleProvider initialLocale="en">
      <NotificationTemplatesTab csrfToken="csrf-token" />
    </LocaleProvider>,
  )
}

describe('NotificationTemplatesTab', () => {
  afterEach(() => restoreGlobals())

  it('lists every key x locale with its customization status', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, listBody)))
    renderTab()

    // 同じ用途が locale ごとに 1 行ずつ並ぶ。
    expect(await screen.findAllByText(t.templatePasswordReset)).toHaveLength(2)
    expect(screen.getByText(t.defaultBadge)).toBeInTheDocument()
    expect(screen.getByText(t.customizedBadge)).toBeInTheDocument()
    expect(screen.getByText(t.localeJa)).toBeInTheDocument()
    expect(screen.getByText(t.localeEn)).toBeInTheDocument()
  })

  it('opens the editor with the allowed placeholders listed', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, detailBody))
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])

    expect(await screen.findByLabelText(t.subjectLabel)).toBeInTheDocument()
    expect(screen.getByText(t.placeholdersHeading)).toBeInTheDocument()
    for (const placeholder of detailBody.placeholders) {
      expect(screen.getByText(`{{${placeholder}}}`)).toBeInTheDocument()
    }
    // 宛先固定は仕様なので UI にも明記する (ADR-142 決定 8)。
    expect(screen.getByText(t.sendTestHelp)).toBeInTheDocument()
  })

  it('saves the edited template as a whole subject/text/html set', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, detailBody))
      .mockResolvedValueOnce(response(200, { ...detailBody, customized: true, subject: 'New' }))
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])
    const subject = await screen.findByLabelText(t.subjectLabel)
    fireEvent.change(subject, { target: { value: 'New' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3))
    const [url, init] = (fetch as any).mock.calls[2]
    expect(url).toBe('/api/admin/v1/tenant/notification_templates/password_reset/ja')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body)).toEqual({
      subject: 'New',
      body_text: detailBody.body_text,
      body_html: detailBody.body_html,
      from_display_name: '',
    })
    expect(await screen.findByText(t.savedNotice)).toBeInTheDocument()
  })

  it('surfaces the server rejection of a disallowed placeholder', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, detailBody))
      .mockResolvedValueOnce(
        response(400, {
          error: 'invalid_notification_template',
          message: 'The template uses a placeholder outside the allowed set.',
        }),
      )
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])
    fireEvent.change(await screen.findByLabelText(t.subjectLabel), {
      target: { value: '{{password}}' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(
      await screen.findByText('The template uses a placeholder outside the allowed set.'),
    ).toBeInTheDocument()
  })

  it('refuses to save when a part of the subject/text/html set is empty', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, detailBody))
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])
    fireEvent.change(await screen.findByLabelText(t.bodyHtmlLabel), { target: { value: '  ' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(await screen.findByText(t.requiredFieldsError)).toBeInTheDocument()
    // 検証で止まるので保存要求は飛ばない。
    expect(fetch).toHaveBeenCalledTimes(2)
  })

  it('renders a preview without sending anything', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, detailBody))
      .mockResolvedValueOnce(
        response(200, {
          subject: 'IdMagic のパスワード再設定',
          body_text: 'Taro Yamada さん https://idp.example.test/reset',
          body_html: '<p>Taro Yamada さん</p>',
        }),
      )
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])
    fireEvent.click(await screen.findByRole('button', { name: t.preview }))

    expect(await screen.findByText(t.previewHeading)).toBeInTheDocument()
    expect(screen.getByText('IdMagic のパスワード再設定')).toBeInTheDocument()
    const [url, init] = (fetch as any).mock.calls[2]
    expect(url).toBe('/api/admin/v1/tenant/notification_templates/password_reset/ja/preview')
    expect(init.method).toBe('POST')
  })

  it('sends a test message without letting the operator choose a recipient', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, detailBody))
      .mockResolvedValueOnce(response(200, { delivered: true, to: 'operator@example.test' }))
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])
    fireEvent.click(await screen.findByRole('button', { name: t.sendTest }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3))
    const [url, init] = (fetch as any).mock.calls[2]
    expect(url).toBe('/api/admin/v1/tenant/notification_templates/password_reset/ja/test')
    expect(init.method).toBe('POST')
    // 宛先を送る手段が無いことを固定する。
    expect(init.body).toBeUndefined()
    expect(
      await screen.findByText(t.sendTestNotice.replace('{to}', 'operator@example.test')),
    ).toBeInTheDocument()
  })

  it('resets the override back to the bundled default', async () => {
    stubGlobal('confirm', mock().mockReturnValue(true))
    const fetch = mock()
      .mockResolvedValueOnce(response(200, listBody))
      .mockResolvedValueOnce(response(200, { ...detailBody, customized: true }))
      .mockResolvedValueOnce(response(200, detailBody))
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click((await screen.findAllByRole('button', { name: t.edit }))[0])
    fireEvent.click(await screen.findByRole('button', { name: t.resetToDefault }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3))
    const [url, init] = (fetch as any).mock.calls[2]
    expect(url).toBe('/api/admin/v1/tenant/notification_templates/password_reset/ja')
    expect(init.method).toBe('DELETE')
    expect(await screen.findByText(t.resetNotice)).toBeInTheDocument()
  })

  // サーバがメッセージを返せばそれを、返せなければ共通の通信失敗文言を出す。空の
  // カタログを黙って表示しない (編集者が「テンプレートが無い」と誤解しないため)。
  it('reports a load failure instead of showing an empty catalog', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(500, {})))
    renderTab()

    expect(await screen.findByText(commonDictionary.en.networkError)).toBeInTheDocument()
    expect(screen.queryByText(t.templatePasswordReset)).not.toBeInTheDocument()
  })

  it('shows the server-supplied message when the catalog cannot be read', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(403, { error: 'access_denied', message: 'Forbidden.' })),
    )
    renderTab()

    expect(await screen.findByText('Forbidden.')).toBeInTheDocument()
  })
})
