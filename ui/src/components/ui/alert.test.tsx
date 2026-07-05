import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Alert } from './alert'

describe('Alert Component', () => {
  it('renders alert with destructive variant by default', () => {
    render(<Alert>Something went wrong</Alert>)
    const alert = screen.getByRole('alert')
    expect(alert).toBeInTheDocument()
    expect(alert).toHaveClass('border-red-200')
    expect(alert).toHaveClass('bg-red-50/80')
    expect(alert).toHaveTextContent('Something went wrong')
  })

  it('renders alert with success variant', () => {
    render(<Alert variant="success">Operation successful</Alert>)
    const alert = screen.getByRole('status')
    expect(alert).toBeInTheDocument()
    expect(alert).toHaveClass('border-emerald-200')
    expect(alert).toHaveClass('bg-emerald-50')
  })

  it('applies custom className', () => {
    render(<Alert className="custom-alert">Message</Alert>)
    const alert = screen.getByRole('alert')
    expect(alert).toHaveClass('custom-alert')
  })
})
