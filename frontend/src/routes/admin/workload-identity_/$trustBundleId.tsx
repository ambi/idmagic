import { createFileRoute, Outlet } from '@tanstack/react-router'

// $trustBundleId は詳細 (index) と編集 (edit) を束ねるレイアウトルート。
export const Route = createFileRoute('/admin/workload-identity_/$trustBundleId')({
  component: Outlet,
})
