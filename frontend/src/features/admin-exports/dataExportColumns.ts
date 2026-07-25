import type { dataExportDictionary } from './DataExportPage.i18n'

// ExportColumn は 1 列の UI 定義。key は backend の allowlist (backend/idmanagement/
// domain/data_export.go) と一致させ、labelKey は i18n dict の key、pii は個人情報を含むか。
export type ExportColumn = {
  key: string
  labelKey: keyof typeof dataExportDictionary.ja
  pii?: boolean
}

// EXPORT_TARGETS は per-type エンドポイントに対応する UI 対象。members は per-group の
// ため list ページには出さず、将来グループ詳細から使う。
export type ExportTarget = 'users' | 'groups'

// backend の allowlist と一致した、種別ごとの選択可能な列。sensitive 値 (password_hash 等)
// は backend にも UI にも存在しない。
export const EXPORT_COLUMNS: Record<ExportTarget, ExportColumn[]> = {
  users: [
    { key: 'id', labelKey: 'colId' },
    { key: 'preferred_username', labelKey: 'colUsername' },
    { key: 'email', labelKey: 'colEmail', pii: true },
    { key: 'name', labelKey: 'colName', pii: true },
    { key: 'given_name', labelKey: 'colGivenName', pii: true },
    { key: 'family_name', labelKey: 'colFamilyName', pii: true },
    { key: 'email_verified', labelKey: 'colEmailVerified' },
    { key: 'mfa_enrolled', labelKey: 'colMfaEnrolled' },
    { key: 'status', labelKey: 'colStatus' },
    { key: 'roles', labelKey: 'colRoles' },
    { key: 'created_at', labelKey: 'colCreatedAt' },
    { key: 'updated_at', labelKey: 'colUpdatedAt' },
  ],
  groups: [
    { key: 'id', labelKey: 'colId' },
    { key: 'name', labelKey: 'colGroupName' },
    { key: 'description', labelKey: 'colDescription' },
    { key: 'membership_type', labelKey: 'colMembershipType' },
    { key: 'roles', labelKey: 'colRoles' },
    { key: 'created_at', labelKey: 'colCreatedAt' },
    { key: 'updated_at', labelKey: 'colUpdatedAt' },
  ],
}
