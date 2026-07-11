import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AdminPaneActions } from './AdminPaneActions'

describe('AdminPaneActions', () => {
  it('renders detail and edit links pointing at the given hrefs', () => {
    render(<AdminPaneActions detailHref="/admin/users/1" editHref="/admin/users/1/edit" />)

    expect(screen.getByRole('link', { name: /詳細/ })).toHaveAttribute('href', '/admin/users/1')
    expect(screen.getByRole('link', { name: /編集/ })).toHaveAttribute(
      'href',
      '/admin/users/1/edit',
    )
  })

  it('invokes onEdit and disables the edit button while busy', () => {
    const onEdit = vi.fn()
    const { rerender } = render(<AdminPaneActions onEdit={onEdit} busy={false} />)

    fireEvent.click(screen.getByRole('button', { name: /編集/ }))
    expect(onEdit).toHaveBeenCalledTimes(1)

    rerender(<AdminPaneActions onEdit={onEdit} busy={true} />)
    expect(screen.getByRole('button', { name: /編集/ })).toBeDisabled()
  })

  it('renders danger-toned secondary actions and respects per-action disabled state', () => {
    const onDelete = vi.fn()
    render(
      <AdminPaneActions
        detailHref="/admin/users/1"
        actions={[
          { label: '削除', onClick: onDelete, tone: 'danger' },
          { label: '無効化', onClick: vi.fn(), disabled: true },
        ]}
      />,
    )

    const deleteButton = screen.getByRole('button', { name: '削除' })
    expect(deleteButton).toHaveClass('text-red-700')
    fireEvent.click(deleteButton)
    expect(onDelete).toHaveBeenCalledTimes(1)

    expect(screen.getByRole('button', { name: '無効化' })).toBeDisabled()
  })

  it('omits the action row entirely when no action is configured', () => {
    render(<AdminPaneActions />)

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})
