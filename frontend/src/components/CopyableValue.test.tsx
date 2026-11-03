import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, mock } from 'bun:test'
import { CopyableValue } from './CopyableValue'

const clipboard = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
afterEach(() => {
  if (clipboard) Object.defineProperty(navigator, 'clipboard', clipboard)
  else Reflect.deleteProperty(navigator, 'clipboard')
})

//spec:covers REQ-IDMANAGEMENT-104: コピーの完了を待ってから成功を表示し、実際の ID を書き込む。
it('reports success only after the clipboard accepts the value', async () => {
  let accept: () => void = () => {}
  const writeText = mock(
    () =>
      new Promise<void>((resolve) => {
        accept = resolve
      }),
  )
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
  render(<CopyableValue value="client-id" copyLabel="Copy ID" />)
  fireEvent.click(screen.getByRole('button', { name: 'Copy ID' }))
  expect(screen.queryByRole('status')?.textContent ?? '').not.toContain('Copied')
  accept()
  await waitFor(() => expect(screen.getByRole('status').textContent).toBe('Copied.'))
  expect(writeText).toHaveBeenCalledWith('client-id')
})

//spec:covers REQ-IDMANAGEMENT-104: コピーの拒否を成功と表示しない。
it('keeps a clipboard failure visible without claiming success', async () => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: {
      writeText: mock(async () => {
        throw new Error('Denied')
      }),
    },
  })
  render(<CopyableValue value="client-id" copyLabel="Copy ID" />)
  fireEvent.click(screen.getByRole('button', { name: 'Copy ID' }))
  await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('Copy failed'))
  expect(screen.queryByRole('status')?.textContent ?? '').not.toContain('Copied')
})
