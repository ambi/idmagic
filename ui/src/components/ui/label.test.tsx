import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Label } from './label'

describe('Label Component', () => {
  it('renders a label element', () => {
    render(<Label>Username</Label>)
    const label = screen.getByText('Username')
    expect(label).toBeInTheDocument()
  })

  it('applies custom className', () => {
    render(<Label className="custom-label">Password</Label>)
    const label = screen.getByText('Password')
    expect(label).toHaveClass('custom-label')
    expect(label).toHaveClass('text-sm')
  })
})
