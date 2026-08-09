import { describe, expect, it, mock } from 'bun:test'
import { render, screen, fireEvent } from '@testing-library/react'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { LoadMoreButton } from './load-more'

const t = commonDictionary.en

describe('LoadMoreButton', () => {
  it('renders nothing when there is no next page', () => {
    const { container } = render(
      <LoadMoreButton hasMore={false} loading={false} onClick={() => {}} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('renders an enabled button when a next page is available', () => {
    render(<LoadMoreButton hasMore loading={false} onClick={() => {}} />)
    const button = screen.getByRole('button')
    expect(button).toBeEnabled()
    expect(button).toHaveTextContent(t.loadMore)
  })

  it('shows a loading label and disables the button while loading', () => {
    render(<LoadMoreButton hasMore loading onClick={() => {}} />)
    const button = screen.getByRole('button')
    expect(button).toBeDisabled()
    expect(button).toHaveTextContent(t.loadingMore)
  })

  it('calls onClick when clicked', () => {
    const onClick = mock()
    render(<LoadMoreButton hasMore loading={false} onClick={onClick} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
