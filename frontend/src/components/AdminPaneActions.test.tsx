import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '../lib/i18n'
import { AdminPaneActions } from './AdminPaneActions'
import { adminPaneActionsDictionary } from './AdminPaneActions.i18n'

const t = adminPaneActionsDictionary.en

function renderEn(ui: ReactElement) {
  return render(<LocaleProvider initialLocale="en">{ui}</LocaleProvider>)
}

describe('AdminPaneActions', () => {
  it('renders detail and edit links pointing at the given hrefs', () => {
    renderEn(<AdminPaneActions detailHref="/admin/users/1" editHref="/admin/users/1/edit" />)

    expect(screen.getByRole('link', { name: new RegExp(t.detail) })).toHaveAttribute(
      'href',
      '/admin/users/1',
    )
    expect(screen.getByRole('link', { name: new RegExp(t.edit) })).toHaveAttribute(
      'href',
      '/admin/users/1/edit',
    )
  })

  it('invokes onEdit and disables the edit button while busy', () => {
    const onEdit = vi.fn()
    const { rerender } = renderEn(<AdminPaneActions onEdit={onEdit} busy={false} />)

    fireEvent.click(screen.getByRole('button', { name: new RegExp(t.edit) }))
    expect(onEdit).toHaveBeenCalledTimes(1)

    rerender(
      <LocaleProvider initialLocale="en">
        <AdminPaneActions onEdit={onEdit} busy={true} />
      </LocaleProvider>,
    )
    expect(screen.getByRole('button', { name: new RegExp(t.edit) })).toBeDisabled()
  })

  it('renders danger-toned secondary actions and respects per-action disabled state', () => {
    const onDelete = vi.fn()
    renderEn(
      <AdminPaneActions
        detailHref="/admin/users/1"
        actions={[
          { label: 'Delete', onClick: onDelete, tone: 'danger' },
          { label: 'Disable', onClick: vi.fn(), disabled: true },
        ]}
      />,
    )

    const deleteButton = screen.getByRole('button', { name: 'Delete' })
    expect(deleteButton).toHaveClass('text-red-700')
    fireEvent.click(deleteButton)
    expect(onDelete).toHaveBeenCalledTimes(1)

    expect(screen.getByRole('button', { name: 'Disable' })).toBeDisabled()
  })

  it('omits the action row entirely when no action is configured', () => {
    renderEn(<AdminPaneActions />)

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})
