import { createFileRoute, Outlet } from '@tanstack/react-router'

// $agentId は詳細 (index) と編集 (edit) を束ねるレイアウトルート。
export const Route = createFileRoute('/admin/agents_/$agentId')({
  component: Outlet,
})
