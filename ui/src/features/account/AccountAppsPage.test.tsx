import { KeyboardSensor, PointerSensor, useSensor, useSensors } from '@dnd-kit/core'
import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AccountAppsPresentation, buildSections } from './AccountAppsPage'
import type { MyApplication, PortalCategory } from '../../types'

const appA: MyApplication = {
  application_id: 'app-a',
  name: 'App A',
  kind: 'weblink',
  category_ids: ['cat-1'],
}
const appB: MyApplication = {
  application_id: 'app-b',
  name: 'App B',
  kind: 'weblink',
  category_ids: [],
  launch_url: 'https://example.com',
}

describe('buildSections', () => {
  const categories: PortalCategory[] = [{ category_id: 'cat-1', name: 'カテゴリ1' }]

  it('groups apps by their assigned category, preserving order', async () => {
    const sections = buildSections([appA, appB], categories)
    expect(sections).toEqual([
      { key: 'cat-1', name: 'カテゴリ1', apps: [appA] },
      { key: '__uncategorized__', name: 'その他', apps: [appB] },
    ])
  })

  it('omits empty sections', async () => {
    const sections = buildSections([appB], categories)
    expect(sections.map((s) => s.key)).toEqual(['__uncategorized__'])
  })

  it('returns no sections for an empty app list', async () => {
    expect(buildSections([], categories)).toEqual([])
  })
})

function Wrapper(props: Partial<Parameters<typeof AccountAppsPresentation>[0]>) {
  const sensors = useSensors(useSensor(PointerSensor), useSensor(KeyboardSensor))
  const base = {
    username: 'taro',
    isAdmin: false,
    order: [appA, appB],
    categories: [],
    activeApp: null,
    saving: false,
    error: null,
    sensors,
    onDragStart: vi.fn(),
    onDragEnd: vi.fn(),
    onDragCancel: vi.fn(),
    onLaunch: vi.fn(),
  }
  return <AccountAppsPresentation {...base} {...props} />
}

describe('AccountAppsPresentation', () => {
  it('shows an empty state when there are no apps', async () => {
    await renderWithRouter(<Wrapper order={[]} />)
    expect(screen.getByText('利用できるアプリはまだありません。')).toBeInTheDocument()
  })

  it('renders app tiles', async () => {
    await renderWithRouter(<Wrapper />)
    expect(screen.getByText('App A')).toBeInTheDocument()
    expect(screen.getByText('App B')).toBeInTheDocument()
  })

  it('shows a saving indicator while persisting order', async () => {
    await renderWithRouter(<Wrapper saving />)
    expect(screen.getByText('並び順を保存中...')).toBeInTheDocument()
  })

  it('shows an error message when present', async () => {
    await renderWithRouter(<Wrapper error="失敗しました" />)
    expect(screen.getByText('失敗しました')).toBeInTheDocument()
  })

  it('calls onLaunch when a launchable tile is clicked', async () => {
    const onLaunch = vi.fn()
    await renderWithRouter(<Wrapper onLaunch={onLaunch} />)
    fireEvent.click(screen.getByText('App B'))
    expect(onLaunch).toHaveBeenCalledWith(appB)
  })
})
