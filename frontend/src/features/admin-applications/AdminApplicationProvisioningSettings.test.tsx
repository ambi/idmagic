import { afterEach, describe, expect, it, mock } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { ConnectionSettingsForm } from './AdminApplicationProvisioningSettings'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import type { AdminGroup, ProvisioningConnection } from '../../types'

const t = provisioningDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const connection: ProvisioningConnection = {
  application_id: 'app-1',
  tenant_id: 'tenant-1',
  status: 'active',
  base_url: 'https://example.com/scim/v2',
  credential: {
    credential_id: 'cred-1',
    auth_method: 'bearer_token',
    created_at: '2026-01-01T00:00:00Z',
  },
  feature_flags: {
    create_users: true,
    update_users: true,
    deactivate_users: true,
    delete_users: false,
    push_groups: false,
  },
  scope: 'assigned_only',
  group_push: null,
  attribute_mappings: [],
  matching: { conflict_match_attribute: 'externalId' },
  deprovision_policy: {
    on_unassign: 'deactivate',
    on_delete: 'deactivate',
    on_group_deleted_or_unassigned: 'none',
    grace_period_days: 0,
    accidental_deletion_count_threshold: null,
    accidental_deletion_percent_threshold: null,
  },
  rate_limit_per_minute: 60,
  max_attempts: 5,
  notification_email: null,
  quarantine_after_consecutive_failures: 5,
  health: 'ok',
  consecutive_failure_count: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const group: AdminGroup = {
  id: 'group-1',
  tenant_id: 'tenant-1',
  name: 'Engineering',
  roles: [],
  member_count: 4,
  created_at: '2026-01-01T00:00:00Z',
}

describe('ConnectionSettingsForm group push consolidation', () => {
  afterEach(() => restoreGlobals())

  it('has a single group-push control that also drives the push_groups feature flag', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, connection)))
    await renderWithRouter(
      <ConnectionSettingsForm
        csrfToken="csrf"
        applicationID="app-1"
        connection={connection}
        onSaved={() => {}}
      />,
    )

    expect(screen.getAllByRole('checkbox', { name: /push groups/i })).toHaveLength(1)

    fireEvent.click(screen.getByLabelText(t.groupPushEnableLabel))
    fireEvent.click(screen.getByRole('button', { name: t.saveButton }))

    await waitFor(() => expect(fetch).toHaveBeenCalled())
    const call = (fetch as unknown as { mock: { calls: unknown[][] } }).mock.calls[0]
    const init = call[1] as RequestInit
    const body = JSON.parse(init.body as string) as { feature_flags: { push_groups: boolean } }
    expect(body.feature_flags.push_groups).toBe(true)
  })

  it('explains the display-name source and selects explicit groups by name', async () => {
    const explicitConnection: ProvisioningConnection = {
      ...connection,
      feature_flags: { ...connection.feature_flags, push_groups: true },
      group_push: {
        selection: 'explicit',
        explicit_group_ids: [],
        display_name_source: 'displayName',
      },
    }
    stubGlobal('fetch', mock().mockResolvedValue(response(200, explicitConnection)))
    await renderWithRouter(
      <ConnectionSettingsForm
        csrfToken="csrf"
        applicationID="app-1"
        connection={explicitConnection}
        groups={[group]}
        onSaved={() => {}}
      />,
    )

    expect(
      screen.getByText('Attribute path whose value is sent as the downstream group display name.'),
    ).toBeInTheDocument()
    const picker = screen.getByRole('combobox', { name: 'Select a group…' })
    fireEvent.mouseDown(picker)
    fireEvent.click(await screen.findByRole('option', { name: group.name }))
    expect(screen.getByText(group.name)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.saveButton }))
    await waitFor(() => expect(fetch).toHaveBeenCalled())
    const call = (fetch as unknown as { mock: { calls: unknown[][] } }).mock.calls[0]
    const init = call[1] as RequestInit
    const body = JSON.parse(init.body as string) as {
      group_push: { explicit_group_ids: string[] }
    }
    expect(body.group_push.explicit_group_ids).toEqual([group.id])
  })
})
