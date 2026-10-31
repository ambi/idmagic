import { afterEach, describe, expect, it, mock, spyOn } from 'bun:test'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import * as api from '../../api'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AgentGuardrailLimits } from '../../types'
import { AgentGuardrailPanel } from './AgentGuardrailPanel'
import { agentGuardrailDictionary } from './AgentGuardrailPanel.i18n'

const t = agentGuardrailDictionary.en

const policy = {
  agent_id: 'bot',
  version: 'version-one-long',
  updated_at: '2026-10-31T00:00:00Z',
  max_token_ttl_seconds: 60,
}

afterEach(() => mock.restore())

describe('AgentGuardrailPanel', () => {
  it('saves against the version it read and omits blank limits', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({ agent: policy })
    const put = spyOn(api, 'putAgentGuardrailPolicy').mockResolvedValue({
      ...policy,
      version: 'version-two',
    })
    await renderWithRouter(<AgentGuardrailPanel agentID="bot" csrfToken="csrf" />)
    await waitFor(() =>
      expect((screen.getByLabelText(t.ttlLabel) as HTMLInputElement).value).toBe('60'),
    )

    fireEvent.change(screen.getByLabelText(t.maxIssuancesLabel), { target: { value: '5' } })
    fireEvent.change(screen.getByLabelText(t.windowSecondsLabel), { target: { value: '60' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(put).toHaveBeenCalled())
    const [, , limits, expected] = put.mock.calls[0] as unknown as [
      string,
      string,
      AgentGuardrailLimits,
      string,
    ]
    expect(expected).toBe('version-one-long')
    expect(limits.issuance_rate).toEqual({ max_issuances: 5, window_seconds: 60 })
    // 空の許可は「何も許さない」なので、空欄から送らない。
    expect(limits.allowed_resources).toBeUndefined()
    await screen.findByText(t.saved)
  })

  it('tells the administrator about a concurrent change', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({})
    spyOn(api, 'putAgentGuardrailPolicy').mockRejectedValue(
      new api.AuthenticationAPIError('conflict', 'agent_guardrail_conflict'),
    )
    await renderWithRouter(<AgentGuardrailPanel agentID="bot" csrfToken="csrf" />)
    fireEvent.click(await screen.findByRole('button', { name: t.save }))
    await screen.findByText(t.conflict)
  })

  it('shows policies without a form when read-only', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({ tenant_default: policy })
    await renderWithRouter(<AgentGuardrailPanel agentID="bot" />)
    await screen.findByText(t.agentNone)
    expect(screen.queryByRole('button', { name: t.save })).toBeNull()
    expect(screen.getByText('60')).toBeTruthy()
  })
})
