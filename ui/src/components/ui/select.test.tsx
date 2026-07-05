import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Select } from './select'

describe('Select Component', () => {
  const options = [
    { value: 'a', label: 'Option A' },
    { value: 'b', label: 'Option B' },
  ]

  it('renders select component with placeholder', () => {
    render(
      <Select value="" onValueChange={vi.fn()} options={options} placeholder="Choose option" />,
    )
    expect(screen.getByText('Choose option')).toBeInTheDocument()
  })

  it('renders active option label', () => {
    render(
      <Select value="b" onValueChange={vi.fn()} options={options} placeholder="Choose option" />,
    )
    expect(screen.getByText('Option B')).toBeInTheDocument()
    expect(screen.queryByText('Choose option')).not.toBeInTheDocument()
  })
})
