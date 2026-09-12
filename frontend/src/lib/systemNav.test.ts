import { describe, it, expect } from 'bun:test'
import { systemNavItems, type SystemNavKey } from './systemNav'

// システムコンソールの並び。テナント横断の操作はここだけに置くので、監査とジョブも
// この一覧に載る (REQ-SYSTEM-020)。
const ORDER: SystemNavKey[] = ['tenants', 'audit-events', 'jobs', 'key-health', 'data-key-health']

describe('systemNavItems', () => {
  it('should list every system console entry in a stable order', () => {
    expect(systemNavItems('tenants').map((item) => item.key)).toEqual(ORDER)
  })

  it('should mark exactly the requested entry active', () => {
    for (const active of ORDER) {
      const items = systemNavItems(active)
      expect(items.filter((item) => item.active).map((item) => item.key)).toEqual([active])
    }
  })

  it('should point the cross-tenant entries at the system console routes', () => {
    const href = (key: SystemNavKey) =>
      systemNavItems('tenants').find((item) => item.key === key)?.href
    expect(href('audit-events')).toBe('/system/audit-events')
    expect(href('jobs')).toBe('/system/jobs')
  })
})
