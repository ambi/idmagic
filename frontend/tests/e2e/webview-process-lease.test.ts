import { expect, test } from 'bun:test'

import { createWebViewProcessLease } from './webview-process-lease'

type FakeView = { name: string; close(): void }

test('the last view is handed to the reset instead of being closed directly', () => {
  const resets: string[] = []
  let firstCloses = 0
  let secondCloses = 0
  const lease = createWebViewProcessLease<FakeView>(async (view) => {
    resets.push(view.name)
  })
  const first = lease.hold({
    name: 'first',
    close: () => {
      firstCloses += 1
    },
  })
  const second = lease.hold({
    name: 'second',
    close: () => {
      secondCloses += 1
    },
  })

  first.close()
  expect(resets).toEqual([])

  second.close()
  second.close()
  expect({ firstCloses, secondCloses, resets }).toEqual({
    firstCloses: 1,
    secondCloses: 0,
    resets: ['second'],
  })
})

test('the next view waits until the reset has finished', async () => {
  let finishReset: (() => void) | undefined
  const lease = createWebViewProcessLease<FakeView>(
    () =>
      new Promise<void>((resolve) => {
        finishReset = resolve
      }),
  )
  const view = lease.hold({ name: 'only', close: () => {} })
  view.close()

  let ready = false
  const waiting = lease.waitUntilReady().then(() => {
    ready = true
  })
  await Bun.sleep(0)
  expect(ready).toBe(false)

  finishReset?.()
  await waiting
  expect(ready).toBe(true)
})
