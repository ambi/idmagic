import { createFileRoute, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/access-reviews_/$planId')({ component: Outlet })
