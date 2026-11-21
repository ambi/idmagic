import {
  listAdminUsers,
  listAdminGroups,
  listAdminAgents,
  listAdminApplications,
} from '../api/admin'
import { listAccessReviewPlans } from '../api/accessReviews'

export async function accessReviewCatalogs() {
  const [users, groups, agents, applications] = await Promise.all([
    listAdminUsers({ all: true }),
    listAdminGroups(),
    listAdminAgents(),
    listAdminApplications(),
  ])
  return { users, groups, agents, applications }
}
export async function accessReviewPlan(id: string) {
  const plan = (await listAccessReviewPlans()).find((item) => item.id === id)
  if (!plan) throw new Error('The review plan was not found.')
  return plan
}
