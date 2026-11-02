import { describe, expect, it } from 'bun:test'
import { render, screen } from '@testing-library/react'
import { UnauthenticatedError } from './api/core'
import { commonDictionary } from './lib/i18n/common.i18n'
import { ErrorScreen } from './router'

const t = commonDictionary.en

describe('ErrorScreen', () => {
  it('does not blame the session for a rendering failure', () => {
    render(<ErrorScreen error={new TypeError('null is not an object')} />)
    expect(screen.getByText(t.pageFailedTitle)).toBeInTheDocument()
    expect(screen.queryByText(t.sessionExpiredMessage)).toBeNull()
    expect(screen.queryByText(t.signInAgain)).toBeNull()
  })

  it('still reports authentication failures as authentication failures', () => {
    render(<ErrorScreen error={new UnauthenticatedError('expired', 'unauthenticated')} />)
    expect(screen.getByText(t.cannotContinueAuthentication)).toBeInTheDocument()
  })
})
