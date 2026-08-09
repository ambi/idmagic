import { expect, mock, test } from 'bun:test'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { PageNavigation } from './page-navigation'

test('PageNavigation exposes all addressable directions and exact position', async () => {
  const onNavigate = mock()
  await renderWithRouter(
    <PageNavigation
      hasFirst
      previousCursor="before"
      nextCursor="after"
      lastCursor="end"
      totalItems={105}
      totalPages={3}
      currentPage={2}
      onNavigate={onNavigate}
    />,
  )

  fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.firstPage }))
  fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.previousPage }))
  fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.nextPage }))
  fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.lastPage }))

  expect(onNavigate).toHaveBeenNthCalledWith(1, null)
  expect(onNavigate).toHaveBeenNthCalledWith(2, 'before')
  expect(onNavigate).toHaveBeenNthCalledWith(3, 'after')
  expect(onNavigate).toHaveBeenNthCalledWith(4, 'end')
  expect(screen.getByText('105 items')).toBeInTheDocument()
  expect(screen.getByText('2 / 3')).toBeInTheDocument()
})

test('PageNavigation keeps empty boundaries visible and disabled', async () => {
  await renderWithRouter(
    <PageNavigation
      hasFirst={false}
      previousCursor={null}
      nextCursor={null}
      lastCursor={null}
      totalItems={0}
      totalPages={0}
      currentPage={0}
      onNavigate={() => {}}
    />,
  )

  expect(screen.getByRole('button', { name: commonDictionary.en.firstPage })).toBeDisabled()
  expect(screen.getByRole('button', { name: commonDictionary.en.previousPage })).toBeDisabled()
  expect(screen.getByRole('button', { name: commonDictionary.en.nextPage })).toBeDisabled()
  expect(screen.getByRole('button', { name: commonDictionary.en.lastPage })).toBeDisabled()
  expect(screen.getByText('0 items')).toBeInTheDocument()
  expect(screen.getByText('0 / 0')).toBeInTheDocument()
})
