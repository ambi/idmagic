import { afterEach, describe, expect, it, mock, spyOn } from 'bun:test'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
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
    expect(await screen.findByRole('status')).toHaveTextContent(t.saved)
    expect(screen.queryByRole('alert')).toBeNull()
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

  //spec:covers WCAG22-LABELS-ERRORS: 契約の範囲を外れる値、片方だけの発行の上限、日時でない値を欄のエラー文で示し、保存しないこと
  it('keeps limits the contract rejects from being saved', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({})
    const put = spyOn(api, 'putAgentGuardrailPolicy')
    await renderWithRouter(<AgentGuardrailPanel agentID="bot" csrfToken="csrf" />)
    const save = await screen.findByRole('button', { name: t.save })

    const depth = screen.getByLabelText(t.depthLabel)
    fireEvent.change(depth, { target: { value: '17' } })
    expect(depth).toHaveAccessibleDescription('Enter a whole number from 0 to 16.')
    fireEvent.change(depth, { target: { value: '16' } })
    expect(depth.getAttribute('aria-invalid')).toBeNull()

    fireEvent.change(screen.getByLabelText(t.maxIssuancesLabel), { target: { value: '5' } })
    const window = screen.getByLabelText(t.windowSecondsLabel)
    fireEvent.blur(window)
    expect(window).toHaveAccessibleDescription('Enter a value for this field.')

    fireEvent.change(window, { target: { value: '60' } })
    const expires = screen.getByLabelText(t.expiresAtLabel)
    fireEvent.change(expires, { target: { value: '2026-02-30T00:00:00Z' } })
    expect(expires).toHaveAccessibleDescription(
      'Enter an RFC 3339 date and time, such as 2026-12-31T23:59:59Z.',
    )
    expect(save).toBeDisabled()
    fireEvent.submit(save.closest('form') as HTMLFormElement)
    expect(put).not.toHaveBeenCalled()

    fireEvent.change(expires, { target: { value: '2026-12-31T23:59:59+09:00' } })
    expect(save).toBeEnabled()
  })

  it('shows policies without a form when read-only', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({ tenant_default: policy })
    await renderWithRouter(<AgentGuardrailPanel agentID="bot" />)
    await screen.findByText(t.agentNone)
    expect(screen.queryByRole('button', { name: t.save })).toBeNull()
    expect(screen.getByText('60')).toBeTruthy()
  })

  it('requires confirmation before deleting and keeps a failure in the dialog', async () => {
    spyOn(api, 'getAgentGuardrailPolicy').mockResolvedValue({ agent: policy })
    const remove = spyOn(api, 'deleteAgentGuardrailPolicy').mockRejectedValue(
      new Error('unavailable'),
    )
    await renderWithRouter(<AgentGuardrailPanel agentID="bot" csrfToken="csrf" />)
    fireEvent.click(await screen.findByRole('button', { name: t.remove }))
    const dialog = await screen.findByRole('dialog')
    expect(remove).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: t.remove }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(t.failed)
    expect(screen.queryByRole('status')).toBeNull()
  })
})
