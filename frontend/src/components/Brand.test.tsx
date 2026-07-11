import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Brand } from './Brand'

describe('Brand Component', () => {
  it('renders brand name and subtitle by default', () => {
    render(<Brand />)
    expect(screen.getByText('IdMagic')).toBeInTheDocument()
    expect(screen.getByText('Identity & Access')).toBeInTheDocument()
  })

  it('renders only brand name when compact is true', () => {
    render(<Brand compact={true} />)
    expect(screen.getByText('IdMagic')).toBeInTheDocument()
    expect(screen.queryByText('Identity & Access')).not.toBeInTheDocument()
  })

  it('applies inverse styling classes', () => {
    const { container } = render(<Brand inverse={true} />)
    // The outermost container has the flex layout
    const brandElement = container.firstChild as HTMLElement
    expect(brandElement).toHaveClass('text-white')
    expect(brandElement).not.toHaveClass('text-slate-950')
  })

  it('renders a same-origin realm branding asset URL as the logo', () => {
    const { container } = render(
      <Brand logoURL="/realms/acme/tenant-branding-assets/logo/logo-key" />,
    )

    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      '/realms/acme/tenant-branding-assets/logo/logo-key',
    )
  })

  it('falls back to the default mark for an unsafe logo URL', () => {
    render(<Brand logoURL="https://evil.example/tenant-branding-assets/logo/logo-key" />)

    expect(document.querySelector('img')).not.toBeInTheDocument()
  })
})
