import { describe, it, expect } from 'bun:test'
import type { ComponentProps } from 'react'
import { render, screen } from '@testing-library/react'
import { Alert } from './alert'

// @ts-expect-error 表示種類の省略を型検査で拒否する。既定値が復活すると、この負例が失敗する。
const missingVariant: ComponentProps<typeof Alert> = { children: 'Saved' }
void missingVariant

describe('Alert Component', () => {
  it('renders an explicitly destructive alert', () => {
    render(<Alert variant="destructive">Something went wrong</Alert>)
    const alert = screen.getByRole('alert')
    expect(alert).toBeInTheDocument()
    expect(alert).toHaveClass('border-red-200')
    expect(alert).toHaveClass('bg-red-50')
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
    render(
      <Alert variant="destructive" className="custom-alert">
        Message
      </Alert>,
    )
    const alert = screen.getByRole('alert')
    expect(alert).toHaveClass('custom-alert')
  })
})
