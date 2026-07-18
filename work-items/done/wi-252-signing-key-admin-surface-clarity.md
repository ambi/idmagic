---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: []
---

# 署名鍵管理画面の運用情報とライフサイクル操作を明確化する

## Motivation

テナント管理者に内部の鍵保管実装を露出すると、運用上の意味が薄いうえに不要な構成情報となる。また、`active` / `verifying` は鍵の状態遷移を誤読させ、現行の active 鍵を無効化できる操作は既発行トークンの検証不能や無署名状態を生む。

## Scope

- `spec/contexts/signing-keys.yaml` の `models.AdminKeyResponse`、`interfaces.DisableTenantKey`、`states.SigningKeyLifecycle`、`scenarios`、`flows`
- tenant admin の署名鍵 API / UI から provider を除去し、system_admin の鍵ヘルス表示だけに残す
- active 鍵の無効化を拒否し、回転後の検証猶予鍵だけを即時無効化可能にする
- 日本語 UI の状態表現を「現在の署名鍵」「移行期間の検証用鍵」に変更する

## Out of Scope

- active 鍵漏洩向けの原子的な緊急ローテート＋失効操作
- KeyProvider 実装・DB schema の変更

## Plan

- SCL で tenant admin response から provider を除き、Disable の precondition を non-active key に限定する。
- domain / adapter で active key の disable を fail-closed にし、HTTP と UI は対象外ボタンを表示しない。
- system_admin health surface の provider は運用診断情報として保持する。

## Tasks

- [x] T001 [SCL] admin contract・状態説明・disable guard・受け入れシナリオを更新して再生成する。
- [x] T002 [Domain/Adapter] RED: `TestInMemoryKeyStoreRejectsDisablingActiveKey` を先に fail 確認（scenario `管理者は回転後の検証用鍵だけを即時無効化できる`）→ GREEN。
- [x] T003 [UI] tenant admin から provider を除き、状態ラベルと無効化対象を明確化する。
- [x] T004 [Verify] Go/UI/SCL 検証を通す。

## Verification

- `just verify-go`
- `just verify-ui`
- `just yaml-check`

## Risk Notes

active 鍵の緊急失効を通常操作から除くため、漏洩対応には別の原子的な専用操作が必要になる。通常の UI から中途半端な disable → rotate を行わせないことを優先する。

## Completion

- **Completed At**: 2026-07-18
- **Summary**: テナント管理者の鍵一覧から provider を除去し、状態を「現在の署名鍵」「移行期間の検証用鍵」と表示した。active 鍵の無効化は拒否し、移行期間の鍵だけを無効化できるようにした。
- **Verification Results**:
  - `just yaml-check-scl` - passed
  - `just scl-render` - passed
  - `just verify-go` - passed
  - `just verify-ui` - passed
- **Affected Guarantees State**: active 鍵を通常の無効化操作で除去しない。Provider は system_admin の鍵ヘルスだけに表示する。
