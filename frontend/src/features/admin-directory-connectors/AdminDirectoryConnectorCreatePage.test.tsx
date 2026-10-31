import { fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminDirectoryConnectorCreatePage } from './AdminDirectoryConnectorCreatePage'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'

const t = adminDirectoryConnectorsDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
  headers: new Headers(),
})

afterEach(() => {
  restoreGlobals()
})

async function submit() {
  await renderWithRouter(<AdminDirectoryConnectorCreatePage csrfToken="csrf" />)
  fireEvent.change(screen.getByLabelText(t.displayNameLabel), { target: { value: 'Corp AD' } })
  fireEvent.click(screen.getByRole('button', { name: t.create }))
}

describe('AdminDirectoryConnectorCreatePage', () => {
  //spec:covers REQ-SOURCING-023: 作成の応答を受けた画面で、遷移せずにトークンを表示すること。
  it('作成するとトークンと詳細へのリンクを表示する', async () => {
    const fetchMock = mock().mockResolvedValue(
      response(201, {
        connector: { id: 'connector/1', display_name: 'Corp AD', status: 'awaiting_enrollment' },
        enrollment_token: 'idmagic_dce_abc',
      }),
    )
    stubGlobal('fetch', fetchMock)
    await submit()

    expect(await screen.findByText('idmagic_dce_abc')).toBeTruthy()
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({ display_name: 'Corp AD' })
    const link = screen.getByText(t.toDetail).closest('a')
    expect(link?.getAttribute('href')).toContain('/admin/directory-connectors/connector%2F1')
    expect(link?.getAttribute('href')).not.toContain('idmagic_dce_abc')
  })

  //spec:covers REQ-SOURCING-023: 拒否された作成はエラーを表示し、トークンを表示しないこと。
  it('拒否されたらエラーを表示し、フォームを残す', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(500)))
    await submit()

    expect(await screen.findByRole('alert')).toBeTruthy()
    expect(screen.queryByText(t.tokenNotice)).toBeNull()
    expect(screen.getByLabelText(t.displayNameLabel)).toBeTruthy()
  })
})
