import { createFileRoute, redirect } from '@tanstack/react-router'

// /system の入口。当面はテナント画面をトップとして送る (将来は専用ダッシュボード)。
export const Route = createFileRoute('/system/')({
  loader: () => {
    throw redirect({ to: '/system/tenants' })
  },
})
