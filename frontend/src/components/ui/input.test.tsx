import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Input } from './input'

describe('Input Component', () => {
  it('renders an input element', () => {
    render(<Input placeholder="Enter name" />)
    const input = screen.getByPlaceholderText('Enter name')
    expect(input).toBeInTheDocument()
    expect(input.tagName).toBe('INPUT')
  })

  it('passes values and handles onChange events', () => {
    const handleChange = vi.fn()
    render(<Input placeholder="Enter name" onChange={handleChange} />)
    const input = screen.getByPlaceholderText('Enter name') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'John' } })
    expect(handleChange).toHaveBeenCalledTimes(1)
    expect(input.value).toBe('John')
  })

  it('supports disabled state', () => {
    render(<Input placeholder="Disabled" disabled />)
    const input = screen.getByPlaceholderText('Disabled')
    expect(input).toBeDisabled()
  })

  it('applies custom className', () => {
    render(<Input placeholder="Custom" className="my-custom-class" />)
    const input = screen.getByPlaceholderText('Custom')
    expect(input).toHaveClass('my-custom-class')
  })
})
