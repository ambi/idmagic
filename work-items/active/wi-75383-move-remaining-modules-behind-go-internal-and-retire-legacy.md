---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on:
  - wi-97546-move-modules-without-private-callers-behind-go-internal
  - wi-33994-reinventory-the-remaining-boundary-debt
  - wi-35767-publish-claim-issuance-as-claimmapping-public-operations
  - wi-60465-publish-job-enqueue-and-handler-registration-as-jobs-ports
  - wi-87746-publish-authentication-operations-other-modules-use
  - wi-93464-move-authorization-login-steps-into-authentication
  - wi-13438-publish-oauth2-consent-revocation-and-client-administration
  - wi-93579-publish-idmanagement-user-operations-and-transactional-writer
  - wi-90942-relocate-saml-assertion-building-out-of-wsfederation
  - wi-28791-publish-application-sign-in-policy-evaluation
change_kind: refactor
spec_impact: { kind: none, reason: "残りのモジュールのパッケージを Go の internal/ へ移し、境界検査から legacy の公開方式を外すだけである。HTTP の応答、認証方式、永続状態、ドメインイベント、外向きの通知は変えない。" }
---

# 残りのモジュールを Go の internal/ へ移し、legacy の公開方式を撤去する

## 動機

[外から非公開パッケージへの import がないモジュールの移行](wi-97546-move-modules-without-private-callers-behind-go-internal.md)の後には、ほかのモジュールから非公開パッケージへ import される 8 個のモジュール（起票時は Tenancy、Authentication、ClaimMapping、Jobs、OAuth2、IdManagement、WsFederation、Application）が `legacy` のまま残る。
これらの依存は[負債の棚卸し](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)が起票する解消の項目と、[テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)で取り除く。
取り除いた後に移す作業と、全モジュールの移行後に `legacy` の命名による判定を消す作業を受け持つ項目がない。

## 対象範囲

- 外から非公開パッケージへの import が解消したモジュールを `internal/` へ移し、責務表の公開方式を `internal` にする。
- 全モジュールが `internal` になった後、境界検査、責務表、構造の文書から `legacy` の公開方式と、組み立て地点からの非公開パッケージへの import を移行期間だけ許す規則を外す。
  公開目的と外側に残る実装を責務表と照合する規則は残す。
- 負債の基準 revision の比較に使う旧形式の互換処理のうち、`legacy` を前提にするものを、基準 revision が新形式になった後で外せるか判断する。

## 対象外

- 外から非公開パッケージへの import そのものの解消。負債の棚卸しが起票する項目とテナントの公開契約の項目で行う。
- モジュールの分割と統合。

## 設計

着手の条件は、対象のモジュールへの `private-import` が台帳から消えていることである。
一部のモジュールの依存だけが先に解消した場合は、そのモジュールから先に移してよい。
全モジュールの移行前に `legacy` を撤去しない。

公開パッケージの決め方は、先行する移行の項目と同じく[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。

## タスク

- [ ] T001 [Inventory] 対象のモジュールへの `private-import` が解消済みかを確かめ、未解消のものの解消の項目を特定する。
- [ ] T002 [App] 解消したモジュールから順に `internal/` へ移し、責務表を `internal` にする。
- [ ] T003 [Tooling] 全モジュールの移行後に `legacy` の判定と移行期間の規則を境界検査から外し、fixture を更新する。
- [ ] T004 [Docs] 責務表と構造の文書から `legacy` の公開方式を外す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 依存の解消が遅れると、この項目が長く着手できない。
  解消したモジュールから移せるようにし、`legacy` の撤去だけを最後に回す。
- `legacy` の撤去と同時に、公開目的の照合まで外してしまうおそれがある。
  撤去するのは命名による判定だけとし、`internal` の照合の fixture を残す。
