import { createFileRoute, Outlet } from '@tanstack/react-router'

// /account/profile は詳細 (index) と編集 (edit) を束ねるレイアウトルート。
export const Route = createFileRoute('/account/profile')({
  component: Outlet,
})
