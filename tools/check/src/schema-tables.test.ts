import { describe, expect, it } from 'bun:test'
import { compareTables, declaredTables, describedTables } from './schema-tables.ts'

const SQL = [
  'CREATE TABLE tenants (',
  '    id UUID PRIMARY KEY,',
  '    realm TEXT NOT NULL',
  ');',
  '',
  'CREATE TABLE tenant_quotas (',
  '    tenant_id UUID PRIMARY KEY,',
  '    users INTEGER NOT NULL',
  ');',
  '',
  'CREATE TABLE users (',
  '    id UUID PRIMARY KEY,',
  '    tenant_id UUID NOT NULL REFERENCES tenants(id)',
  ');',
  '',
  'CREATE TABLE scim_user_refs (',
  '    tenant_id UUID NOT NULL,',
  '    scim_id TEXT NOT NULL,',
  '    -- tenant_id を含む主キーにする理由は設計文書にある。',
  '    PRIMARY KEY (tenant_id, scim_id)',
  ');',
  '',
  'CREATE TABLE group_members (',
  '    group_id UUID NOT NULL,',
  '    user_id UUID NOT NULL,',
  '    PRIMARY KEY (group_id, user_id)',
  ');',
  '',
  'CREATE UNLOGGED TABLE login_counters (',
  '    tenant_id UUID NOT NULL,',
  '    key_hash TEXT NOT NULL,',
  '    PRIMARY KEY (tenant_id, key_hash)',
  ') WITH (fillfactor = 80);',
].join('\n')

const DOC = [
  '# データベース設計',
  '',
  '| テーブル | 役割 | 所有 Context | テーブル種別 | `tenant_id` 列 |',
  '| --- | --- | --- | --- | --- |',
  '| `tenants` | テナントそのもの | Tenancy | `LOGGED` | なし |',
  '| `tenant_quotas` | 上限 | Tenancy | `LOGGED` | 単独主キー |',
  '| `users` | 利用者 | IdManagement | `LOGGED` | 非キー列 |',
  '',
  '本文の表は対象にしない。',
  '',
  '| 区分 | 説明 |',
  '| --- | --- |',
  '| `not_a_table` | 見出しが「テーブル」ではない |',
  '',
  '| テーブル | 役割 | 所有 Context | テーブル種別 | `tenant_id` 列 |',
  '| --- | --- | --- | --- | --- |',
  '| `scim_user_refs` | 外部 ID の対応 | Sourcing | `LOGGED` | 複合主キーの一部 |',
  '| `group_members` | 所属 | IdManagement | `LOGGED` | なし |',
  '| `login_counters` | 回数 | Authentication | `UNLOGGED` | 複合主キーの一部 |',
].join('\n')

describe('declaredTables', () => {
  it('reads persistence and tenant_id placement from each CREATE TABLE', () => {
    expect(declaredTables(SQL)).toEqual([
      { name: 'tenants', unlogged: false, tenantId: 'absent' },
      { name: 'tenant_quotas', unlogged: false, tenantId: 'primary-key' },
      { name: 'users', unlogged: false, tenantId: 'column' },
      { name: 'scim_user_refs', unlogged: false, tenantId: 'primary-key-part' },
      { name: 'group_members', unlogged: false, tenantId: 'absent' },
      { name: 'login_counters', unlogged: true, tenantId: 'primary-key-part' },
    ])
  })

  it('does not read a tenant_id mentioned in an SQL comment as a column', () => {
    const tables = declaredTables(
      ['CREATE TABLE notes (', '    -- tenant_id は持たない', '    id UUID PRIMARY KEY', ');'].join(
        '\n',
      ),
    )
    expect(tables[0]?.tenantId).toBe('absent')
  })
})

describe('describedTables', () => {
  it('collects rows only from tables whose first header is テーブル', () => {
    expect(describedTables(DOC).map((table) => table.name)).toEqual([
      'tenants',
      'tenant_quotas',
      'users',
      'scim_user_refs',
      'group_members',
      'login_counters',
    ])
  })

  it('records the source line and the classification cells', () => {
    expect(describedTables(DOC)[0]).toEqual({
      name: 'tenants',
      line: 5,
      kind: '`LOGGED`',
      tenantId: 'なし',
    })
  })
})

describe('compareTables', () => {
  const declared = declaredTables(SQL)

  it('accepts a description that matches the schema', () => {
    expect(compareTables(declared, describedTables(DOC))).toEqual([])
  })

  it('reports a table the schema declares but no row describes', () => {
    const described = describedTables(DOC).filter((table) => table.name !== 'users')
    expect(compareTables(declared, described).map((finding) => finding.message)).toEqual([
      'users is declared in the schema but not described',
    ])
  })

  it('reports a row for a table the schema does not declare', () => {
    const described = describedTables(DOC.replace('`group_members`', '`group_memberships`'))
    expect(compareTables(declared, described).map((finding) => finding.message)).toEqual([
      'group_members is declared in the schema but not described',
      'line 18: group_memberships is described but not declared in the schema',
    ])
  })

  it('reports a table described twice', () => {
    const described = describedTables(
      `${DOC}\n| \`users\` | 重複 | IdManagement | \`LOGGED\` | 非キー列 |`,
    )
    expect(compareTables(declared, described).map((finding) => finding.message)).toEqual([
      'line 20: users is described more than once',
    ])
  })

  it('reports a logged table described as UNLOGGED and the reverse', () => {
    const described = describedTables(
      DOC.replace(
        '| Tenancy | `LOGGED` | 単独主キー |',
        '| Tenancy | `UNLOGGED` | 単独主キー |',
      ).replace('| `UNLOGGED` | 複合主キーの一部 |', '| `LOGGED` | 複合主キーの一部 |'),
    )
    expect(compareTables(declared, described).map((finding) => finding.message)).toEqual([
      'line 6: tenant_quotas is a logged table but is described as `UNLOGGED`',
      'line 19: login_counters is an UNLOGGED table but is described as `LOGGED`',
    ])
  })

  it('reports a tenant_id classification that disagrees with the schema', () => {
    const described = describedTables(
      DOC.replace('| `LOGGED` | 非キー列 |', '| `LOGGED` | なし |').replace(
        '| Sourcing | `LOGGED` | 複合主キーの一部 |',
        '| Sourcing | `LOGGED` | 非キー列 |',
      ),
    )
    expect(compareTables(declared, described).map((finding) => finding.message)).toEqual([
      'line 7: users has tenant_id as 非キー列 but is described as なし',
      'line 17: scim_user_refs has tenant_id as 複合主キーの一部 but is described as 非キー列',
    ])
  })

  it('reports an empty schema as a finding rather than a silent pass', () => {
    expect(compareTables([], []).map((finding) => finding.message)).toEqual([
      'the schema declares no table',
    ])
  })
})
