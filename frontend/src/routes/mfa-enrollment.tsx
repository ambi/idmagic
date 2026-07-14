import { createFileRoute } from '@tanstack/react-router'
import { BrowserFlowRoute, loadBrowserFlowData } from './-authFlow'

export const Route = createFileRoute('/mfa-enrollment')({
  loader: ({ location }) => loadBrowserFlowData('/mfa-enrollment', location.searchStr),
  component: MfaEnrollmentRoute,
})

function MfaEnrollmentRoute() {
  return <BrowserFlowRoute data={Route.useLoaderData()} />
}
