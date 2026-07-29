import { createFileRoute } from '@tanstack/react-router'
import { HomePage } from '../features/auth-flow/HomePage'
import { PageMarker } from './-page'

export const Route = createFileRoute('/')({
  loader: () => ({
    demoEnabled: import.meta.env.DEV || import.meta.env.VITE_DEMO_LOGIN_ENABLED === 'true',
  }),
  component: HomeRoute,
})

function HomeRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="home">
      <HomePage {...data} />
    </PageMarker>
  )
}
