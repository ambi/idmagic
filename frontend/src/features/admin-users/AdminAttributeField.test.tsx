import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { UserAttributeDef } from '../../types'
import { AdminAttributeField } from './AdminUserAttributeEditor'

function def(overrides: Partial<UserAttributeDef> = {}): UserAttributeDef {
  return {
    key: 'department',
    label: 'Department',
    type: 'string',
    multi_valued: false,
    required: false,
    editable_by_user: true,
    visibility: 'admin_readable',
    pii: false,
    ...overrides,
  }
}

describe('AdminAttributeField', () => {
  it('renders a checkbox for boolean attributes and reports toggles', () => {
    const onChange = vi.fn()
    render(<AdminAttributeField def={def({ type: 'boolean' })} value="false" onChange={onChange} />)
    const checkbox = screen.getByRole('checkbox', { name: /Department/ })
    expect(checkbox).not.toBeChecked()
    fireEvent.click(checkbox)
    expect(onChange).toHaveBeenCalledWith('true')
  })

  it('ignores checkbox toggles while read-only', () => {
    const onChange = vi.fn()
    render(
      <AdminAttributeField
        def={def({ type: 'boolean' })}
        value="false"
        onChange={onChange}
        readOnly
      />,
    )
    fireEvent.click(screen.getByRole('checkbox', { name: /Department/ }))
    expect(onChange).not.toHaveBeenCalled()
  })

  it('renders a number input for number attributes', () => {
    render(<AdminAttributeField def={def({ type: 'number' })} value="3" onChange={vi.fn()} />)
    const input = screen.getByLabelText(/Department/) as HTMLInputElement
    expect(input.type).toBe('number')
  })

  it('renders a date input for date attributes', () => {
    render(<AdminAttributeField def={def({ type: 'date' })} value="" onChange={vi.fn()} />)
    const input = screen.getByLabelText(/Department/) as HTMLInputElement
    expect(input.type).toBe('date')
  })

  it('renders a comma-separated placeholder for string_array attributes', () => {
    render(<AdminAttributeField def={def({ type: 'string_array' })} value="" onChange={vi.fn()} />)
    const input = screen.getByLabelText(/Department/) as HTMLInputElement
    expect(input.type).toBe('text')
    expect(input.placeholder).not.toBe('')
  })

  it('reports text changes and marks read-only fields', () => {
    const onChange = vi.fn()
    render(<AdminAttributeField def={def()} value="Engineering" onChange={onChange} readOnly />)
    const input = screen.getByLabelText(/Department/) as HTMLInputElement
    expect(input).toHaveAttribute('readonly')
    fireEvent.change(input, { target: { value: 'Sales' } })
    expect(onChange).toHaveBeenCalledWith('Sales')
  })
})
