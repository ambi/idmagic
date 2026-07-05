import { createFileRoute, Outlet } from '@tanstack/react-router'

// $groupId は詳細 (index) と編集 (edit) を束ねるレイアウトルート。
export const Route = createFileRoute('/admin/groups_/$groupId')({
  component: Outlet,
})
