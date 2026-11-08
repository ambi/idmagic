import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { BindingForm } from './BindingForm'
import type { AdminAgent } from '../../types'

const t = adminWorkloadIdentityDictionary.en
const agent: AdminAgent = {
  id: 'agent',
  tenant_id: 'tenant',
  name: 'Worker',
  kind: 'autonomous',
  owner_user_id: 'owner',
  status: 'active',
  client_ids: ['client'],
  created_at: '2026-11-08T00:00:00Z',
}
const overlap = {
  id: 'existing',
  tenant_id: 'tenant',
  trust_bundle_id: 'bundle',
  subject_pattern: 'ns/*',
  agent_id: 'agent',
  status: 'enabled',
  created_at: '2026-11-08T00:00:00Z',
}
afterEach(restoreGlobals)

//spec:covers REQ-WORKLOADIDENTITY-016: 重なり候補を保存の前に表示し、再操作で保存を許す。
it('requires review of an overlap before saving and checks again after a pattern change', async () => {
  const onSave = mock(async () => {})
  stubGlobal(
    'fetch',
    mock(async () => ({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ matches: [], possible_overlaps: [overlap] }),
    })),
  )
  await renderWithRouter(
    <BindingForm
      csrfToken="csrf"
      trustBundleID="bundle"
      agents={[agent]}
      onSave={onSave}
      saveLabel={t.addBinding}
    />,
  )
  fireEvent.change(screen.getByLabelText(t.subjectPatternLabel), { target: { value: 'ns/worker' } })
  fireEvent.submit(screen.getByRole('button', { name: t.addBinding }).closest('form')!)
  await screen.findByText(t.overlapWarning)
  expect(onSave).not.toHaveBeenCalled()
  await waitFor(() =>
    expect(screen.getByRole('button', { name: t.saveWithOverlap })).not.toBeDisabled(),
  )
  fireEvent.click(screen.getByRole('button', { name: t.saveWithOverlap }))
  await waitFor(() =>
    expect(onSave).toHaveBeenCalledWith({ subject_pattern: 'ns/worker', agent_id: 'agent' }),
  )
  await waitFor(() => expect(screen.getByLabelText(t.subjectPatternLabel)).toHaveValue(''))
  fireEvent.change(screen.getByLabelText(t.subjectPatternLabel), { target: { value: 'ns/other' } })
  fireEvent.click(screen.getByRole('button', { name: t.addBinding }))
  await screen.findByText(t.overlapWarning)
  expect(onSave).toHaveBeenCalledTimes(1)
})

//spec:covers REQ-WORKLOADIDENTITY-016: 試行の失敗は入力を保ち、保存を実行しない。
it('keeps input and reports a preview failure without saving', async () => {
  const onSave = mock(async () => {})
  stubGlobal(
    'fetch',
    mock(async () => ({
      ok: false,
      status: 400,
      headers: new Headers(),
      json: async () => ({ detail: 'Invalid pattern' }),
    })),
  )
  await renderWithRouter(
    <BindingForm
      csrfToken="csrf"
      trustBundleID="bundle"
      agents={[agent]}
      onSave={onSave}
      saveLabel={t.addBinding}
    />,
  )
  fireEvent.change(screen.getByLabelText(t.subjectPatternLabel), { target: { value: '[' } })
  fireEvent.click(screen.getByRole('button', { name: t.addBinding }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Invalid pattern')
  expect(screen.getByLabelText(t.subjectPatternLabel)).toHaveValue('[')
  expect(onSave).not.toHaveBeenCalled()
})
