import { describe, it, expect, mock } from 'bun:test'
import { render, screen, fireEvent } from '@testing-library/react'
import { Button } from './button'

describe('Button Component', () => {
  it('renders a button with default styles and text', () => {
    render(<Button>Click me</Button>)
    const button = screen.getByRole('button', { name: 'Click me' })
    expect(button).toBeInTheDocument()
    expect(button).toHaveClass('bg-primary') // Default variant class
    expect(button).toHaveClass('h-9') // Default size class
  })

  it('applies variant classes correctly', () => {
    render(<Button variant="destructive">Delete</Button>)
    const button = screen.getByRole('button', { name: 'Delete' })
    expect(button).toHaveClass('bg-destructive/10')
  })

  it('applies size classes correctly', () => {
    render(<Button size="lg">Large</Button>)
    const button = screen.getByRole('button', { name: 'Large' })
    expect(button).toHaveClass('h-10')
  })

  it('calls onClick when clicked', async () => {
    const handleClick = mock()
    render(<Button onClick={handleClick}>Click</Button>)
    const button = screen.getByRole('button', { name: 'Click' })
    fireEvent.click(button)
    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('does not call onClick when disabled', async () => {
    const handleClick = mock()
    render(
      <Button onClick={handleClick} disabled>
        Click
      </Button>,
    )
    const button = screen.getByRole('button', { name: 'Click' })
    expect(button).toBeDisabled()
    fireEvent.click(button)
    expect(handleClick).not.toHaveBeenCalled()
  })

  it('renders as a custom element when render is given', () => {
    render(
      <Button nativeButton={false} render={<a href="/test" />}>
        Link Button
      </Button>,
    )
    // Base UI gives a non-<button> render target role="button" (matching its visual/
    // interactive presentation), so it's queried as a button, not a link.
    const link = screen.getByRole('button', { name: 'Link Button' })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', '/test')
    expect(link).toHaveClass('bg-primary')
  })
})
