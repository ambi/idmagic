import { expect, test } from 'bun:test'

import { createWebViewProcessLease } from './webview-process-lease'

test('the last view delegates its close to the shared browser process reset', () => {
  let resets = 0
  let firstCloses = 0
  let secondCloses = 0
  const lease = createWebViewProcessLease(() => {
    resets += 1
  })
  const first = lease.hold({
    close: () => {
      firstCloses += 1
    },
  })
  const second = lease.hold({
    close: () => {
      secondCloses += 1
    },
  })

  first.close()
  expect(resets).toBe(0)

  second.close()
  expect(resets).toBe(1)

  second.close()
  expect({ firstCloses, secondCloses, resets }).toEqual({
    firstCloses: 1,
    secondCloses: 0,
    resets: 1,
  })
})

test('the next view waits until the shared browser process has settled', async () => {
  let finishSettling: (() => void) | undefined
  const lease = createWebViewProcessLease(
    () => {},
    () =>
      new Promise<void>((resolve) => {
        finishSettling = resolve
      }),
  )
  const view = lease.hold({ close: () => {} })
  view.close()

  let ready = false
  const waiting = lease.waitUntilReady().then(() => {
    ready = true
  })
  await Bun.sleep(0)
  expect(ready).toBe(false)

  finishSettling?.()
  await waiting
  expect(ready).toBe(true)
})
