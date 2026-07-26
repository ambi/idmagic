import { createFileRoute, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/settings_/saml-idp-profiles_/$profileId')({
  component: Outlet,
})
