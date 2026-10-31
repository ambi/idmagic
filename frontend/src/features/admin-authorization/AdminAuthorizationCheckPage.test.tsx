import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AdminAuthorizationCheckPage } from './AdminAuthorizationCheckPage'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'

const t = adminAuthorizationDictionary.en
const originalFetch = globalThis.fetch

function stubResponse(status: number, body: unknown) {
  const requests: { url: string; body: unknown }[] = []
  globalThis.fetch = mock(async (input: RequestInfo | URL, init?: RequestInit) => {
    requests.push({ url: String(input), body: JSON.parse(String(init?.body)) })
    return new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': status === 200 ? 'application/json' : 'application/problem+json' },
    })
  }) as unknown as typeof fetch
  return requests
}

async function renderFilled(consistency = '') {
  await renderWithRouter(
    <AdminAuthorizationCheckPage csrfToken="csrf" resourceTypes={[]} consistency={consistency} />,
  )
  fireEvent.change(screen.getByLabelText(t.subjectType), { target: { value: 'user' } })
  fireEvent.change(screen.getByLabelText(t.subjectId), { target: { value: 'u1' } })
  fireEvent.change(screen.getByLabelText(t.resourceType), { target: { value: 'doc' } })
  fireEvent.change(screen.getByLabelText(t.relation), { target: { value: 'viewer' } })
}

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe('AdminAuthorizationCheckPage', () => {
  it('shows the relation path in order for a permitted check', async () => {
    const requests = stubResponse(200, {
      result: {
        permitted: true,
        model_version: 7,
        consistency: 'c',
        relation_path: ['doc#viewer', 'folder#viewer'],
        reasons: [],
      },
    })
    await renderFilled('token-from-write')
    fireEvent.change(screen.getByLabelText(t.resourceId), { target: { value: 'd1' } })
    fireEvent.click(screen.getByRole('button', { name: t.runCheck }))

    await screen.findByText(t.permitted)
    const path = within(screen.getByRole('list', { name: t.relationPath }))
      .getAllByRole('listitem')
      .map((item) => item.textContent)
    expect(path).toEqual(['doc#viewer', 'folder#viewer'])
    expect(requests[0]?.body).toEqual({
      resource_type: 'doc',
      resource_id: 'd1',
      relation: 'viewer',
      subject_type: 'user',
      subject_id: 'u1',
      minimum_consistency: 'token-from-write',
    })
  })

  //spec:covers REQ-AUTHORIZATION-013: 既知の理由を説明し、辞書にない理由をコードのまま示す。
  it('explains known reasons and shows unknown reasons as they are', async () => {
    stubResponse(200, {
      result: {
        permitted: false,
        model_version: 7,
        consistency: 'c',
        relation_path: [],
        reasons: ['relationship_permits_actor_chain', 'constructor', 'future_rule'],
      },
    })
    await renderFilled()
    fireEvent.change(screen.getByLabelText(t.resourceId), { target: { value: 'd1' } })
    fireEvent.click(screen.getByRole('button', { name: t.runCheck }))

    await screen.findByText(t.denied)
    expect(screen.getByText(t.reasonRelationshipPermitsActorChain)).toBeInTheDocument()
    expect(screen.getByText('constructor')).toBeInTheDocument()
    expect(screen.getByText('future_rule')).toBeInTheDocument()
  })

  it('sends the actor chain and scopes', async () => {
    const requests = stubResponse(200, {
      result: {
        permitted: false,
        model_version: 1,
        consistency: 'c',
        relation_path: [],
        reasons: [],
      },
    })
    await renderFilled()
    fireEvent.change(screen.getByLabelText(t.resourceId), { target: { value: 'd1' } })
    fireEvent.click(screen.getByRole('button', { name: t.addActor }))
    fireEvent.change(screen.getByLabelText(t.actorType), { target: { value: 'agent' } })
    fireEvent.change(screen.getByLabelText(t.actorId), { target: { value: 'a1' } })
    fireEvent.change(screen.getByLabelText(t.grantedScopes), { target: { value: 'read  write' } })
    fireEvent.change(screen.getByLabelText(t.requiredScopes), { target: { value: 'read' } })
    fireEvent.click(screen.getByRole('button', { name: t.runCheck }))
    await waitFor(() => expect(requests).toHaveLength(1))
    expect(requests[0]?.body).toMatchObject({
      actor_chain: [{ type: 'agent', id: 'a1' }],
      granted_scopes: ['read', 'write'],
      required_scopes: ['read'],
    })
  })

  it('enumerates without a resource id and warns about truncation', async () => {
    const requests = stubResponse(200, {
      result: { resource_ids: ['d1', 'd2'], truncated: true, model_version: 2, consistency: 'c' },
    })
    await renderFilled()
    expect(screen.getByRole('button', { name: t.runCheck })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: t.runList }))
    await screen.findByText('d2')
    expect(screen.getByText(t.listTruncated)).toBeInTheDocument()
    expect(requests[0]?.url).toContain('/api/admin/v1/authorization/list-accessible-resources')
  })

  it('does not warn about truncation for a complete enumeration', async () => {
    stubResponse(200, {
      result: { resource_ids: [], truncated: false, model_version: 2, consistency: 'c' },
    })
    await renderFilled()
    fireEvent.click(screen.getByRole('button', { name: t.runList }))
    await screen.findByText(t.listEmpty)
    expect(screen.queryByText(t.listTruncated)).not.toBeInTheDocument()
  })

  it('explains an unsatisfied consistency token', async () => {
    stubResponse(422, {
      type: 'urn:idmagic:error:consistency_not_satisfied',
      title: 'Unprocessable',
      status: 422,
      detail: 'The store has not caught up with the supplied consistency token.',
    })
    await renderFilled('stale')
    fireEvent.change(screen.getByLabelText(t.resourceId), { target: { value: 'd1' } })
    fireEvent.click(screen.getByRole('button', { name: t.runCheck }))
    await screen.findByText(t.consistencyNotSatisfied)
  })
})
