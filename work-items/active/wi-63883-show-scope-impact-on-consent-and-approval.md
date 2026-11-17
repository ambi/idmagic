---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-11-17
priority: p2
group: federation-protocols
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-008 }
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-057 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-043 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-051 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-052 }
---

# 資源とスコープに影響度を登録し、同意と承認の画面に示す

## 動機

同意と承認の画面は、スコープ名と `authorization_details` を示すが、その操作が何を変えるかを示さない。
利用者は「読み取りだけ」か「取り消せない変更」かを区別できずに Agent へ許可を与える。

[Poppy 1.0](https://personalagentprotocol.org/docs/1.0/spec) §4.7 は、スコープごとに `changes`（`none`、`reversible`、`irreversible`）と `personal_data` の手がかりを企業に必須とし、Personal Agent は未知の値を `irreversible` として扱う。
[[wi-10861-support-poppy-authorization-server-profile]] の前提になり、Poppy 以外の同意と承認の質も上げるので、群の基準の p3 より上げて p2 とする。

## 対象範囲

- スコープ、`authorization_details` の種類、MCP のツールに、変更の有無（なし、取り消せる、取り消せない）と個人データの扱いを登録する管理 API と管理画面。
- 同意画面、Agent への代行の付与画面、承認画面での表示。
- Poppy の `scope_hints` へ写せる形での公開。公開する場所（`poppy.json` か AS メタデータか）は wi-10861 で決める。

## 対象外

- 影響度に応じた認証水準の要求。[[wi-88709-require-assurance-for-irreversible-approvals]] が扱う。
- 影響度による自動の許可や拒否。表示と判断の材料にだけ使う。
- 部分的なスコープ選択の UI。

## 設計

影響度は登録者の申告であり、IdMagic が業務の事実を保証するとは表示しない。
未登録の値は画面で「影響度未登録」と示し、取り消せない操作と同等に注意を促す。
Poppy の Personal Agent が未知の値を `irreversible` として扱うのと揃える。

値の語彙は Poppy と同じ三値に合わせる。
独自の段階を足す案は、外部へ写すときに変換の規則が要り、取り違えの原因になるので採らない。
一度公開した値の意味は変えられないので `irreversible` とする。

## 計画

1. 語彙と所有者（OAuth2 の資源とスコープの登録）を確定し、仕様と TypeSpec を更新する。
2. 管理 API、表示、公開の順に実装する。

## タスク

- [ ] T001 [Spec] 影響度の語彙、登録、表示の規範と TypeSpec を書く。
- [ ] T002 [Acceptance] 登録、表示、未登録の扱いの RED を作る。
- [ ] T003 [Adapters] 管理 API と保存を実装する。
- [ ] T004 [UI] 同意、付与、承認の画面に表示する。
- [ ] T005 [Verify] 検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`。
- `mise run check-spec`、`mise run check-schema`。
- 画面は対象の `test-ui-e2e-file`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>`。

## リスク

誤って「変更なし」と登録すると、利用者は安心して危険な操作を許可する。
登録の変更を監査に残し、未登録より緩い表示を既定にしない。
