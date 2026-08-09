import { expect, mock, test } from 'bun:test'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { PageNavigation } from './page-navigation'

test('PageNavigation exposes both addressable directions', async () => {
  const onNavigate = mock()
  await renderWithRouter(
    <PageNavigation previousCursor="before" nextCursor="after" onNavigate={onNavigate} />,
  )

  fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.previousPage }))
  fireEvent.click(screen.getByRole('button', { name: commonDictionary.en.nextPage }))

  expect(onNavigate).toHaveBeenNthCalledWith(1, 'before')
  expect(onNavigate).toHaveBeenNthCalledWith(2, 'after')
})

test('PageNavigation hides directions that do not exist', async () => {
  await renderWithRouter(
    <PageNavigation previousCursor={null} nextCursor={null} onNavigate={() => {}} />,
  )

  expect(screen.queryByRole('button', { name: commonDictionary.en.previousPage })).toBeNull()
  expect(screen.queryByRole('button', { name: commonDictionary.en.nextPage })).toBeNull()
})
