import { createFileRoute, Outlet } from '@tanstack/react-router'
import { AdminTheme } from '../../components/AdminTheme'

export const Route = createFileRoute('/admin')({
  component: AdminLayout,
})

function AdminLayout() {
  return (
    <AdminTheme>
      <Outlet />
    </AdminTheme>
  )
}
