import { describe, expect, it } from 'bun:test'
import { act, renderHook, waitFor } from '@testing-library/react'
import { usePaginatedList } from './usePaginatedList'

describe('usePaginatedList', () => {
  it('starts from the initial page', () => {
    const { result } = renderHook(() =>
      usePaginatedList<number>({ items: [1, 2], nextCursor: 'c1' }),
    )
    expect(result.current.items).toEqual([1, 2])
    expect(result.current.hasMore).toBe(true)
    expect(result.current.loadingMore).toBe(false)
  })

  it('hasMore is false when the initial page has no next cursor', () => {
    const { result } = renderHook(() => usePaginatedList<number>({ items: [1], nextCursor: null }))
    expect(result.current.hasMore).toBe(false)
  })

  it('appends the next page and advances the cursor', async () => {
    const { result } = renderHook(() =>
      usePaginatedList<number>({ items: [1, 2], nextCursor: 'c1' }),
    )
    await act(async () => {
      await result.current.loadMore(async (cursor) => {
        expect(cursor).toBe('c1')
        return { items: [3, 4], nextCursor: null }
      })
    })
    expect(result.current.items).toEqual([1, 2, 3, 4])
    expect(result.current.hasMore).toBe(false)
  })

  it('does not call fetchPage when there is no next cursor', async () => {
    const { result } = renderHook(() => usePaginatedList<number>({ items: [1], nextCursor: null }))
    let called = false
    await act(async () => {
      await result.current.loadMore(async () => {
        called = true
        return { items: [], nextCursor: null }
      })
    })
    expect(called).toBe(false)
  })

  it('sets loadingMore while the fetch is in flight and clears it after', async () => {
    const { result } = renderHook(() => usePaginatedList<number>({ items: [1], nextCursor: 'c1' }))
    let resolveFetch: (() => void) | undefined
    const pending = new Promise<void>((resolve) => {
      resolveFetch = resolve
    })
    let loadMorePromise!: Promise<void>
    act(() => {
      loadMorePromise = result.current.loadMore(async () => {
        await pending
        return { items: [2], nextCursor: null }
      })
    })
    await waitFor(() => expect(result.current.loadingMore).toBe(true))
    resolveFetch?.()
    await act(async () => {
      await loadMorePromise
    })
    expect(result.current.loadingMore).toBe(false)
  })

  it('rethrows fetch errors and clears loadingMore', async () => {
    const { result } = renderHook(() => usePaginatedList<number>({ items: [1], nextCursor: 'c1' }))
    let caught: unknown
    await act(async () => {
      try {
        await result.current.loadMore(async () => {
          throw new Error('boom')
        })
      } catch (error) {
        caught = error
      }
    })
    expect(caught).toBeInstanceOf(Error)
    expect((caught as Error).message).toBe('boom')
    expect(result.current.loadingMore).toBe(false)
  })

  it('reset replaces items and cursor', () => {
    const { result } = renderHook(() =>
      usePaginatedList<number>({ items: [1, 2], nextCursor: 'c1' }),
    )
    act(() => {
      result.current.reset({ items: [9], nextCursor: null })
    })
    expect(result.current.items).toEqual([9])
    expect(result.current.hasMore).toBe(false)
  })
})
