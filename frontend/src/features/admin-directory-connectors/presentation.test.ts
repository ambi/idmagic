import { describe, expect, it } from 'bun:test'
import type { DirectoryConnector, IngestionRunDetail } from '../../types'
import { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'
import { canManage, enrollmentExpired, hiddenFailureCount, itemErrorLabel } from './presentation'

const t = adminDirectoryConnectorsDictionary.en

const connector = (overrides: Partial<DirectoryConnector>): DirectoryConnector => ({
  id: 'c1',
  display_name: 'AD',
  status: 'awaiting_enrollment',
  created_at: '2026-10-31T00:00:00Z',
  accidental_deletion_count_threshold: 500,
  accidental_deletion_percent_threshold: 20,
  ...overrides,
})

const run = (failed: number, listed: number | null): IngestionRunDetail => ({
  id: 'r1',
  mode: 'full',
  status: 'completed',
  started_at: '2026-10-31T00:00:00Z',
  counts: {
    batches: 1,
    items: failed,
    created: 0,
    updated: 0,
    unchanged: 0,
    deleted: 0,
    failed,
    disabled: 0,
  },
  failures:
    listed === null
      ? null
      : Array.from({ length: listed }, (_, i) => ({
          batch_id: 'b',
          external_id: String(i),
          error: 'quota_exceeded',
        })),
})

describe('enrollmentExpired', () => {
  const expires = '2026-10-31T01:00:00Z'
  //spec:covers REQ-SOURCING-024: 有効期限の時刻そのものから期限切れとして表示すること。
  it('有効期限の直前は期限内、有効期限の時刻から期限切れとする', () => {
    const c = connector({ enrollment_expires_at: expires })
    expect(enrollmentExpired(c, new Date('2026-10-31T00:59:59.999Z'))).toBe(false)
    expect(enrollmentExpired(c, new Date(expires))).toBe(true)
  })

  it('登録待ちでなければ期限切れとしない', () => {
    const now = new Date('2026-11-01T00:00:00Z')
    expect(
      enrollmentExpired(connector({ status: 'active', enrollment_expires_at: expires }), now),
    ).toBe(false)
    expect(enrollmentExpired(connector({}), now)).toBe(false)
  })
})

describe('canManage', () => {
  it('無効の Connector だけ操作を出さない', () => {
    expect(canManage(connector({ status: 'awaiting_enrollment' }))).toBe(true)
    expect(canManage(connector({ status: 'active' }))).toBe(true)
    expect(canManage(connector({ status: 'disabled' }))).toBe(false)
  })
})

describe('hiddenFailureCount', () => {
  it('上限で切れた分だけを数える', () => {
    expect(hiddenFailureCount(run(100, 100))).toBe(0)
    expect(hiddenFailureCount(run(101, 100))).toBe(1)
    expect(hiddenFailureCount(run(3, null))).toBe(3)
  })
})

describe('itemErrorLabel', () => {
  it('既知のコードを説明にし、未知のコードを隠さない', () => {
    expect(itemErrorLabel('quota_exceeded', t)).toBe(t.itemErrorQuotaExceeded)
    expect(itemErrorLabel('new_code', t)).toBe('new_code')
  })
})
