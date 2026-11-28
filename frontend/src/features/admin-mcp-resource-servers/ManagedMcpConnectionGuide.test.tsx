import { screen } from '@testing-library/react'
import { expect, test } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { ManagedMcpConnectionGuide } from './ManagedMcpConnectionGuide'

//spec:covers REQ-OAUTH2-100: 本人の連携が必要な構成だけに、本人向けの入口とコネクタを示す。
test('managed vault guide points users to their own connections', async () => {
  const t = adminMcpResourceServersDictionary.en
  const server: McpResourceServer = {
    id: 's1',
    tenant_id: 'example',
    name: 'Documents',
    resource: 'https://idmagic.example/realms/example/mcp/s1',
    scopes: ['read'],
    state: 'active',
    created_at: '2026-11-29T00:00:00Z',
    updated_at: '2026-11-29T00:00:00Z',
    hosting: {
      upstream_url: 'https://upstream.example/mcp',
      authentication: 'token_vault',
      connector_id: 'documents-connector',
    },
  }
  await renderWithRouter(<ManagedMcpConnectionGuide server={server} />)
  expect(screen.getByRole('link', { name: t.connectionServices }).getAttribute('href')).toMatch(
    /\/account\/connections$/,
  )
  expect(screen.getByText(`${t.connectorID}: documents-connector`)).toBeInTheDocument()
  expect(screen.getByText(server.resource)).toBeInTheDocument()
  expect(screen.queryByText(server.hosting!.upstream_url)).toBeNull()
})
