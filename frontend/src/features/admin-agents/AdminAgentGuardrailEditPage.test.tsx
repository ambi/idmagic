import { afterEach, describe, expect, it, mock, spyOn } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import * as api from '../../api'
import { renderWithRouter } from '../../test/renderWithRouter'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import type { AdminAgent } from '../../types'
import { AdminAgentGuardrailEditPage } from './AdminAgentGuardrailEditPage'
import { agentGuardrailDictionary } from './AgentGuardrailPanel.i18n'
import { takeAgentNotice } from './agentNavigation'

const t = agentGuardrailDictionary.en
const agent: AdminAgent = {
  id: 'bot',
  tenant_id: 'default',
  name: 'Bot',
  kind: 'autonomous',
  owner_user_id: 'owner',
  status: 'active',
  roles: [],
  client_ids: [],
  created_at: '2026-01-01T00:00:00Z',
}
const policy = {
  agent_id: 'bot',
  version: 'v1',
  updated_at: '2026-01-01T00:00:00Z',
  max_token_ttl_seconds: 60,
}

afterEach(() => {
  mock.restore()
  restoreGlobals()
  sessionStorage.clear()
})

describe('AdminAgentGuardrailEditPage', () => {
  it('returns to the detail only after saving and delivers a success notice once', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({ agent: policy })
    const put = spyOn(api, 'putAgentGuardrailPolicy').mockResolvedValue({
      ...policy,
      version: 'v2',
    })
    const assign = mock()
    stubGlobal('location', { ...window.location, assign })
    await renderWithRouter(<AdminAgentGuardrailEditPage csrfToken="csrf" agent={agent} />)
    fireEvent.click(await screen.findByRole('button', { name: t.save }))
    await waitFor(() => expect(assign).toHaveBeenCalledWith('/admin/agents/bot'))
    expect(put.mock.calls[0]?.[3]).toBe('v1')
    expect(takeAgentNotice('other')).toBe('')
    expect(takeAgentNotice('bot')).toBe(t.saved)
    expect(takeAgentNotice('bot')).toBe('')
  })

  it('keeps edits on a conflict and does not deliver a success notice', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({ agent: policy })
    spyOn(api, 'putAgentGuardrailPolicy').mockRejectedValue(
      new api.AuthenticationAPIError('conflict', 'agent_guardrail_conflict'),
    )
    const assign = mock()
    stubGlobal('location', { ...window.location, assign })
    await renderWithRouter(<AdminAgentGuardrailEditPage csrfToken="csrf" agent={agent} />)
    await screen.findByRole('button', { name: t.save })
    fireEvent.change(screen.getByLabelText(t.ttlLabel), { target: { value: '120' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))
    expect(await screen.findByRole('alert')).toHaveTextContent(t.conflict)
    expect(screen.getByLabelText(t.ttlLabel)).toHaveValue('120')
    expect(assign).not.toHaveBeenCalled()
    expect(takeAgentNotice('bot')).toBe('')
  })

  it('can cancel without sending the draft', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({})
    const put = spyOn(api, 'putAgentGuardrailPolicy')
    await renderWithRouter(<AdminAgentGuardrailEditPage csrfToken="csrf" agent={agent} />)
    const cancel = await screen.findByRole('button', { name: t.cancel })
    fireEvent.change(screen.getByLabelText(t.ttlLabel), { target: { value: '120' } })
    expect(cancel).toHaveAttribute('href', '/admin/agents/bot')
    expect(put).not.toHaveBeenCalled()
    expect(screen.getAllByRole('button', { name: t.save })).toHaveLength(1)
  })
})
