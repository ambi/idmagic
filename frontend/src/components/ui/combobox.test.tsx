import { fireEvent, screen } from '@testing-library/react'
import { describe, expect, it, mock } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import { SearchableSelect } from './combobox'

describe('SearchableSelect', () => {
  it('uses the placeholder as its accessible name and selects a filtered option', async () => {
    const onValueChange = mock()
    await renderWithRouter(
      <SearchableSelect
        value=""
        onValueChange={onValueChange}
        placeholder="Select a group…"
        options={[
          { value: 'group-1', label: 'Engineering' },
          { value: 'group-2', label: 'Sales' },
        ]}
      />,
    )

    const input = screen.getByRole('combobox', { name: 'Select a group…' })
    fireEvent.mouseDown(input)
    fireEvent.change(input, { target: { value: 'Eng' } })

    fireEvent.click(await screen.findByRole('option', { name: 'Engineering' }))
    expect(screen.queryByRole('option', { name: 'Sales' })).not.toBeInTheDocument()
    expect(onValueChange).toHaveBeenCalledWith('group-1')
  })
})
