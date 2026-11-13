import { createFileRoute, Outlet } from '@tanstack/react-router'

// $realm はテナントの詳細 (index) と各変更画面を束ねるレイアウトルート。
export const Route = createFileRoute('/system/tenants_/$realm')({
  component: Outlet,
})
