import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Card } from './card'

describe('Card Component', () => {
  it('renders card content', () => {
    render(<Card data-testid="card">Card Content</Card>)
    const card = screen.getByTestId('card')
    expect(card).toBeInTheDocument()
    expect(card).toHaveTextContent('Card Content')
    expect(card).toHaveClass('rounded-lg')
  })

  it('applies custom className', () => {
    render(<Card className="my-card">Card</Card>)
    const card = screen.getByText('Card')
    expect(card).toHaveClass('my-card')
  })
})
