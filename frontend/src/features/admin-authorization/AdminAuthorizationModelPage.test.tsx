import { screen } from '@testing-library/react'
import { describe, expect, it } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AuthorizationModel } from '../../types'
import { AdminAuthorizationModelPage } from './AdminAuthorizationModelPage'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'

const t = adminAuthorizationDictionary.en

const model: AuthorizationModel = {
  id: 'm1',
  tenant_id: 'default',
  version: 3,
  resource_types: [
    {
      name: 'doc',
      relations: [
        {
          name: 'viewer',
          rewrites: [
            { kind: 'direct', direct_subject_types: ['user', 'group#member'] },
            { kind: 'computed_userset', computed_relation: 'editor' },
            { kind: 'tuple_to_userset', tupleset_relation: 'parent', computed_relation: 'reader' },
          ],
        },
      ],
    },
  ],
  created_at: '2026-01-01T00:00:00Z',
}

describe('AdminAuthorizationModelPage', () => {
  it('expands the rewrites into readable sentences', async () => {
    await renderWithRouter(<AdminAuthorizationModelPage model={model} />)
    expect(
      screen.getByText(t.ruleDirect.replace('{forms}', 'user, group#member')),
    ).toBeInTheDocument()
    expect(screen.getByText(t.ruleComputed.replace('{relation}', 'editor'))).toBeInTheDocument()
    expect(
      screen.getByText(
        t.ruleTupleToUserset.replace('{tupleset}', 'parent').replace('{relation}', 'reader'),
      ),
    ).toBeInTheDocument()
  })

  it('announces the publication made on the previous page', async () => {
    await renderWithRouter(<AdminAuthorizationModelPage model={model} justPublished />)
    expect(screen.getByRole('status')).toHaveTextContent(t.publishedNotice)
  })

  it('offers publication when no model exists', async () => {
    await renderWithRouter(<AdminAuthorizationModelPage model={null} />)
    expect(screen.getByText(t.noModel)).toBeInTheDocument()
    expect(screen.getByText(t.editModel).closest('a')).toHaveAttribute(
      'href',
      '/admin/authorization/model/edit',
    )
  })

  it('marks an old version and links to the previous and latest versions', async () => {
    await renderWithRouter(<AdminAuthorizationModelPage model={model} requestedVersion={3} />)
    expect(screen.getByText(t.pastVersion.replace('{version}', '3'))).toBeInTheDocument()
    expect(screen.getByRole('link', { name: t.previousVersion }).getAttribute('href')).toContain(
      'version=2',
    )
    expect(screen.getByRole('link', { name: t.latestVersion })).toBeInTheDocument()
  })

  it('reports a requested version that does not exist', async () => {
    await renderWithRouter(<AdminAuthorizationModelPage model={null} requestedVersion={9} />)
    expect(screen.getByText(t.versionNotFound.replace('{version}', '9'))).toBeInTheDocument()
  })
})
