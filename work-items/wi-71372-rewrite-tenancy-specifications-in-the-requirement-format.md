---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-39141-move-tenancy-to-the-feature-and-design-layout, wi-86874-model-test-identity-management-state-machines-from-their-matrices]
change_kind: docs
affected_spec:
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-002 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-020 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-043 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-004 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-005 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-032 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-033 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-034 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-035 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-001 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-041 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-042 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-003 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-014 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-025 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-026 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-027 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-015 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-016 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-017 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-018 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-038 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-039 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-040 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-012 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-013 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-036 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-006 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-007 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-008 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-009 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-010 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-011 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-022 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-023 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-024 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-019 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-021 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-028 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-029 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-030 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-031 }
---

# Tenancy の機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

## 動機

wi-39141 は Tenancy の文書を機能仕様と内部設計の構造へ移すが、規則の内容と書き方は変えない。
wi-17076 で定めた書き方（EARS 形式の要件、要約表と担保手段と要判断の廃止、状態遷移表（マトリクス形式））は、構成の移行の後に別に適用する必要がある。
Tenancy の文書は約 1,300 行あり、IdManagement に次いで大きい。

## 対象範囲

- wi-39141 で移した Tenancy の機能仕様を、`user` と同じ方式で書き直す。
  要件の ID は変えず、本文を EARS 形式に改める。
- テナントのライフサイクルの状態機械に状態遷移表（マトリクス形式）を加え、wi-86874 の方式でモデルベースのテストを加える。
- `check-unspecified-vocabulary` の対象に Tenancy を加え、導入時点の違反を許容リストに載せてから、書き直しで消す。
- 書き直しで見つけた未記載の振る舞いを、仕様にない振る舞いの分類に従って扱い、(c) は起票する。

## 対象外

- 構成の移行。
  wi-39141 が扱う。
- (c) に分類した振る舞いの実装の変更。

## タスク

- [ ] T001 [Spec] 機能仕様を書き直す。
- [ ] T002 [Spec] 状態遷移表（マトリクス形式）を加える。
- [ ] T003 [Test] モデルベースのテストを加える。
- [ ] T004 [Tooling] 語彙の検査の対象に Tenancy を加え、許容リストを減らす。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-unspecified-vocabulary`
- `mise run spec-diff` で、要件の差分が本文の書き直しと、状態遷移表の追加だけであること。
- `mise run verify`

## リスク

- テナントの解決とリソース上限は、セキュリティの境界に関わる。
  拒否の条件を述べる文を落とさないよう、書き直しの前後で義務を突き合わせる。
