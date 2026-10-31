import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AuthorizationModel, ResourceTypeDefinition } from '../../types'
import { AdminAuthorizationModelEditPage } from './AdminAuthorizationModelEditPage'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'

const t = adminAuthorizationDictionary.en
const originalFetch = globalThis.fetch

const types: ResourceTypeDefinition[] = [
  { name: 'user', relations: [] },
  {
    name: 'doc',
    relations: [
      { name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['user', 'user:*'] }] },
    ],
  },
]
const model: AuthorizationModel = {
  id: 'm1',
  tenant_id: 'default',
  version: 3,
  resource_types: types,
  created_at: '2026-01-01T00:00:00Z',
}

function json(status: number, body: unknown, contentType = 'application/json') {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': contentType } })
}

function stubAPI(
  latest: AuthorizationModel = model,
  publish = () => json(201, { authorization_model: model, consistency: 'c2' }),
) {
  const posts: unknown[] = []
  globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
    if (init?.method === 'POST') {
      posts.push(JSON.parse(String(init.body)))
      return publish()
    }
    return json(200, { authorization_model: latest, consistency: 'c1' })
  }) as unknown as typeof fetch
  return posts
}

afterEach(() => {
  globalThis.fetch = originalFetch
})

async function renderPage(onPublished = () => {}) {
  await renderWithRouter(
    <AdminAuthorizationModelEditPage csrfToken="csrf" model={model} onPublished={onPublished} />,
  )
}

function openJson() {
  fireEvent.click(screen.getByRole('button', { name: t.modeJson }))
}

function setJson(value: ResourceTypeDefinition[] | string) {
  fireEvent.change(screen.getByLabelText(t.jsonLabel), {
    target: { value: typeof value === 'string' ? value : JSON.stringify(value) },
  })
}

describe('AdminAuthorizationModelEditPage', () => {
  //spec:covers REQ-AUTHORIZATION-011: 主体の形の削除を差分と警告で示し、確定するまで登録を要求しない。
  it('shows the removed subject form and publishes only after confirmation', async () => {
    const posts = stubAPI()
    const onPublished = mock(() => {})
    await renderPage(onPublished)
    openJson()
    const narrowed: ResourceTypeDefinition[] = [
      types[0] as ResourceTypeDefinition,
      {
        name: 'doc',
        relations: [
          { name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['user'] }] },
        ],
      },
    ]
    setJson(narrowed)
    fireEvent.click(screen.getByRole('button', { name: t.review }))

    await screen.findByText(t.diffNarrowingWarning)
    expect(screen.getByText(/direct user:\*/)).toBeInTheDocument()
    expect(posts).toEqual([])

    fireEvent.click(screen.getByRole('button', { name: t.publish }))
    await waitFor(() => expect(onPublished).toHaveBeenCalled())
    expect(posts).toEqual([{ resource_types: narrowed }])
  })

  it('compares with the latest version read when reviewing', async () => {
    stubAPI({
      ...model,
      version: 4,
      resource_types: [...types, { name: 'added_by_someone_else', relations: [] }],
    })
    await renderPage()
    fireEvent.click(screen.getByRole('button', { name: t.review }))
    await screen.findByText(/added_by_someone_else/)
    expect(screen.getByText(t.diffNarrowingWarning)).toBeInTheDocument()
  })

  it('does not offer publication when nothing changed', async () => {
    stubAPI()
    await renderPage()
    fireEvent.click(screen.getByRole('button', { name: t.review }))
    await screen.findByText(t.diffEmpty)
    expect(screen.getByRole('button', { name: t.publish })).toBeDisabled()
  })

  it('keeps the JSON mode and the text when the JSON cannot return to the form', async () => {
    stubAPI()
    await renderPage()
    openJson()
    setJson('{broken')
    fireEvent.click(screen.getByRole('button', { name: t.modeForm }))
    expect(screen.getByText(t.jsonInvalid)).toBeInTheDocument()
    expect(screen.getByLabelText(t.jsonLabel)).toHaveValue('{broken')
    expect(screen.getByRole('button', { name: t.review })).toBeDisabled()
  })

  it('keeps the same model across a round trip between the modes', async () => {
    stubAPI()
    await renderPage()
    openJson()
    fireEvent.click(screen.getByRole('button', { name: t.modeForm }))
    openJson()
    expect(JSON.parse((screen.getByLabelText(t.jsonLabel) as HTMLTextAreaElement).value)).toEqual(
      types,
    )
  })

  it('builds the three subject forms from choices without writing strings', async () => {
    stubAPI()
    await renderPage()
    fireEvent.click(screen.getByRole('button', { name: t.addRewrite }))
    const prefix = 'type-1-relation-0-rewrite-1'
    const choose = (suffix: string, value: string) =>
      fireEvent.change(document.getElementById(`${prefix}-${suffix}`) as HTMLElement, {
        target: { value },
      })
    const add = () => {
      const buttons = screen.getAllByRole('button', { name: t.addSubjectForm })
      fireEvent.click(buttons[buttons.length - 1] as HTMLElement)
    }
    choose('type', 'user')
    add()
    choose('kind', 'subject_set')
    choose('type', 'doc')
    choose('relation', 'viewer')
    add()
    choose('kind', 'wildcard')
    choose('type', 'user')
    add()

    openJson()
    const edited = JSON.parse((screen.getByLabelText(t.jsonLabel) as HTMLTextAreaElement).value)
    expect(edited[1].relations[0].rewrites[1]).toEqual({
      kind: 'direct',
      direct_subject_types: ['user', 'doc#viewer', 'user:*'],
    })
  })

  it('marks an invalid name and blocks the review', async () => {
    stubAPI()
    await renderPage()
    fireEvent.click(screen.getByRole('button', { name: t.addResourceType }))
    const names = screen.getAllByLabelText(t.resourceTypeName)
    fireEvent.change(names[names.length - 1] as HTMLElement, { target: { value: 'Invalid' } })
    expect(names[names.length - 1]).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('button', { name: t.review })).toBeDisabled()
  })

  it('keeps the input and shows the detail when the server refuses the model', async () => {
    stubAPI(model, () =>
      json(
        422,
        {
          type: 'urn:idmagic:error:authorization_model_invalid',
          title: 'Unprocessable',
          status: 422,
          detail: 'authorization model invalid: computed_userset cycle through a#r',
        },
        'application/problem+json',
      ),
    )
    await renderPage()
    openJson()
    setJson([{ name: 'added', relations: [] }, ...types])
    fireEvent.click(screen.getByRole('button', { name: t.review }))
    await screen.findByText(t.diffTitle)
    fireEvent.click(screen.getByRole('button', { name: t.publish }))
    await screen.findByText(/computed_userset cycle/)
    expect((screen.getByLabelText(t.jsonLabel) as HTMLTextAreaElement).value).toContain('added')
  })
})
