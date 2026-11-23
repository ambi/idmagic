import { expect, test } from 'bun:test'
import { screen } from '@testing-library/react'
import { ConsentPage } from './ConsentPage'
import { renderWithRouter } from '../../test/renderWithRouter'

//spec:covers REQ-OAUTH2-098: 未登録のスコープについても許可する前に影響度と個人データの不明を注意表示する。
test('consent shows unknown permission impact before allowing', async () => {
  await renderWithRouter(
    <ConsentPage csrfToken="csrf" clientName="Agent" scopes={['unknown:scope']} />,
  )
  expect(screen.getByText('Impact not registered')).toBeTruthy()
  expect(screen.getByText('Personal data handling unknown')).toBeTruthy()
  expect(screen.getByText('Review as an irreversible change.')).toBeTruthy()
  expect(
    screen.getByText(
      'These are claims made by the registrant. IdMagic does not verify their business effects.',
    ),
  ).toBeTruthy()
})

//spec:covers REQ-OAUTH2-098: 登録したスコープと認可詳細の三値と個人データの扱いをそれぞれ示す。
test('consent shows registered scope and detail impacts', async () => {
  await renderWithRouter(
    <ConsentPage
      csrfToken="csrf"
      clientName="Agent"
      scopes={['read']}
      scopeHints={{ read: { changes: 'none', personal_data: false } }}
      authorizationDetails={[
        {
          type: 'payment',
          summary: 'Pay',
          impact: { changes: 'irreversible', personal_data: true },
        },
      ]}
    />,
  )
  expect(screen.getByText('No changes')).toBeTruthy()
  expect(screen.getByText('Does not handle personal data')).toBeTruthy()
  expect(screen.getByText('Irreversible changes')).toBeTruthy()
  expect(screen.getByText('Handles personal data')).toBeTruthy()
})
