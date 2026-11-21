import { adminRequest, request } from './core'

export type ReviewDecision = 'retain' | 'revoke'
export type ReviewTargetKind = 'group_membership' | 'application_assignment' | 'agent'
export type AccessReviewPlanInput = {
  name: string
  target_kind: ReviewTargetKind
  resource_ids: string[]
  reviewer_id?: string
  fallback_reviewer_id: string
  default_decision: ReviewDecision
  starts_at: string
  duration_days: number
  recurrence_days: number
  enabled: boolean
}
export type AccessReviewPlan = AccessReviewPlanInput & {
  id: string
  revision: number
  next_start_at?: string
  last_error: string
}
export type AccessReviewItem = {
  id: string
  resource_id: string
  resource_name: string
  subject_type: string
  subject_id: string
  subject_name: string
  reviewer_id: string
  decision: '' | ReviewDecision
  decided_by: string
  decided_at?: string
  defaulted: boolean
  outcome: string
  error_code: string
}
export type AccessReview = {
  id: string
  plan_id: string
  name: string
  target_kind: ReviewTargetKind
  default_decision: ReviewDecision
  status: 'open' | 'applying' | 'completed'
  started_at: string
  due_at: string
  notification_failed: boolean
  items: AccessReviewItem[]
}
export async function listAccessReviewPlans() {
  return (await request<{ plans: AccessReviewPlan[] }>('/api/admin/v1/access-review-plans')).plans
}
export function createAccessReviewPlan(csrf: string, input: AccessReviewPlanInput) {
  return request<AccessReviewPlan>(
    '/api/admin/v1/access-review-plans',
    adminRequest(csrf, 'POST', input),
  )
}
export function updateAccessReviewPlan(
  csrf: string,
  plan: AccessReviewPlan,
  input: AccessReviewPlanInput,
) {
  const {
    name,
    target_kind,
    resource_ids,
    reviewer_id,
    fallback_reviewer_id,
    default_decision,
    starts_at,
    duration_days,
    recurrence_days,
    enabled,
  } = input
  return request<AccessReviewPlan>(
    `/api/admin/v1/access-review-plans/${encodeURIComponent(plan.id)}`,
    adminRequest(csrf, 'PUT', {
      name,
      target_kind,
      resource_ids,
      reviewer_id,
      fallback_reviewer_id,
      default_decision,
      starts_at,
      duration_days,
      recurrence_days,
      enabled,
      expected_revision: plan.revision,
    }),
  )
}
export async function listAccessReviews(planID: string) {
  return (
    await request<{ reviews: AccessReview[] }>(
      `/api/admin/v1/access-review-plans/${encodeURIComponent(planID)}/reviews`,
    )
  ).reviews
}
export async function listMyAccessReviews() {
  return (await request<{ reviews: AccessReview[] }>('/api/account/v1/access-reviews')).reviews
}
export function decideAccessReview(
  csrf: string,
  reviewID: string,
  itemID: string,
  decision: ReviewDecision,
) {
  return request<AccessReview>(
    `/api/account/v1/access-reviews/${encodeURIComponent(reviewID)}/items/${encodeURIComponent(itemID)}`,
    adminRequest(csrf, 'POST', { decision }),
  )
}
