import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, jest, spyOn } from 'bun:test'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AccessReview, AccessReviewPlan } from '../../api/accessReviews'
import type { AdminGroup, AdminUser } from '../../types'
import {
  AccessReviewPlanEditor,
  AccessReviewPlanPage,
  MyAccessReviewsPage,
} from './AccessReviewPages'

afterEach(() => jest.restoreAllMocks())

const user: AdminUser = {
  id: 'reviewer',
  preferred_username: '確認担当',
  roles: [],
  email_verified: true,
  mfa_enrolled: false,
  status: 'active',
  created_at: '2026-11-22T00:00:00Z',
  updated_at: '2026-11-22T00:00:00Z',
}
const group: AdminGroup = {
  id: 'team',
  tenant_id: 'tenant',
  name: '営業チーム',
  roles: [],
  member_count: 1,
  membership_type: 'manual',
  created_at: '2026-11-22T00:00:00Z',
}
const plan: AccessReviewPlan = {
  id: 'plan',
  revision: 1,
  name: '四半期の見直し',
  target_kind: 'group_membership',
  resource_ids: ['team'],
  reviewer_id: 'reviewer',
  fallback_reviewer_id: 'reviewer',
  default_decision: 'revoke',
  starts_at: '2099-11-22T00:00:12.345678Z',
  duration_days: 14,
  recurrence_days: 30,
  enabled: true,
  last_error: '',
}
const review: AccessReview = {
  id: 'review',
  plan_id: 'plan',
  name: plan.name,
  target_kind: 'group_membership',
  default_decision: 'revoke',
  status: 'open',
  started_at: '2026-11-22T00:00:00Z',
  due_at: '2099-12-01T00:00:00Z',
  notification_failed: false,
  items: [
    {
      id: 'item',
      resource_id: 'team',
      resource_name: '営業チーム',
      subject_type: 'user',
      subject_id: 'user',
      subject_name: '利用者',
      reviewer_id: 'reviewer',
      decision: '',
      decided_by: '',
      defaulted: false,
      outcome: '',
      error_code: '',
    },
  ],
}

//spec:covers REQ-IDGOVERNANCE-026: 動的と上流管理の所属は選ばせず、未回答時の失効は確認の前に送信しない。拒否しても入力を保つ。
it('計画の対象を限定し、未回答時の失効を確認してから保存する', async () => {
  const request = spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response(
      JSON.stringify({ detail: 'Access denied', type: 'urn:idmagic:error:access_denied' }),
      { status: 403, headers: { 'Content-Type': 'application/problem+json' } },
    ),
  )
  await renderWithRouter(
    <AccessReviewPlanEditor
      csrfToken="csrf"
      initial={plan}
      users={[user]}
      groups={[
        group,
        { ...group, id: 'dynamic', name: '自動の所属', membership_type: 'dynamic' },
        { ...group, id: 'upstream', name: '外部の所属', upstream_managed: true },
      ]}
      agents={[]}
      applications={[]}
    />,
    { locale: 'ja' },
  )
  expect(screen.queryByLabelText('自動の所属')).not.toBeInTheDocument()
  expect(screen.queryByLabelText('外部の所属')).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '保存' }))
  expect(request).not.toHaveBeenCalled()
  expect(screen.getByRole('dialog')).toHaveTextContent('各回の開始後は処置を変更できません')
  fireEvent.click(screen.getByRole('button', { name: '確定する' }))
  await screen.findByRole('alert')
  expect(screen.getByLabelText('名前')).toHaveValue(plan.name)
  expect(request).toHaveBeenCalledTimes(1)
  const body = JSON.parse(String(request.mock.calls[0]?.[1]?.body))
  expect(body).toMatchObject({
    resource_ids: ['team'],
    default_decision: 'revoke',
    expected_revision: 1,
    starts_at: plan.starts_at,
  })
  expect(body).not.toHaveProperty('last_error')
})

//spec:covers REQ-IDGOVERNANCE-026: 失効は対象を確認して送信し、失敗時に未回答を保ち、成功時だけ判断と成功通知を更新する。
it('本人の回答は失敗を成功に見せず、確認後に失効を確定する', async () => {
  let fail = true
  const request = spyOn(globalThis, 'fetch').mockImplementation(
    Object.assign(
      async (path: RequestInfo | URL) => {
        if (String(path) === '/api/branding') return Response.json({})
        return fail
          ? new Response(
              JSON.stringify({
                type: 'urn:idmagic:error:access_review_conflict',
                detail: 'Review expired',
              }),
              { status: 409, headers: { 'Content-Type': 'application/problem+json' } },
            )
          : Response.json({
              ...review,
              items: [{ ...review.items[0]!, decision: 'revoke', decided_by: 'reviewer' }],
            })
      },
      { preconnect: fetch.preconnect },
    ),
  )
  await renderWithRouter(
    <MyAccessReviewsPage csrfToken="csrf" username="確認担当" isAdmin={false} reviews={[review]} />,
    { locale: 'ja' },
  )
  fireEvent.click(screen.getByRole('button', { name: '失効と回答' }))
  expect(screen.getByRole('dialog')).toHaveTextContent('営業チーム / 利用者')
  expect(request.mock.calls.filter(([path]) => String(path).includes('/items/'))).toEqual([])
  fireEvent.click(screen.getByRole('button', { name: '確定する' }))
  await waitFor(() => expect(screen.getAllByRole('alert').length).toBeGreaterThan(0))
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
  fail = false
  fireEvent.click(screen.getByRole('button', { name: '確定する' }))
  await screen.findByRole('status')
  expect(screen.queryByRole('button', { name: '失効と回答' })).not.toBeInTheDocument()
  expect(screen.getByText('判断: 失効')).toBeInTheDocument()
})

//spec:covers REQ-IDGOVERNANCE-026: 計画の詳細を読み取り専用にし、判断と適用を別に表示する。
it('結果は未回答時の判断と適用結果を区別する', async () => {
  await renderWithRouter(
    <AccessReviewPlanPage
      csrfToken="csrf"
      plan={plan}
      reviews={[
        {
          ...review,
          status: 'completed',
          items: [{ ...review.items[0]!, decision: 'revoke', defaulted: true, outcome: 'revoked' }],
        },
      ]}
    />,
    { locale: 'ja' },
  )
  expect(screen.queryByLabelText('名前')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: '計画を編集' })).toHaveAttribute(
    'href',
    '/admin/access-reviews/plan/edit',
  )
  expect(screen.getByText('失効済み')).toBeInTheDocument()
  expect(screen.getByRole('columnheader', { name: '判断' })).toBeInTheDocument()
  expect(screen.getByRole('columnheader', { name: '適用結果' })).toBeInTheDocument()
})
