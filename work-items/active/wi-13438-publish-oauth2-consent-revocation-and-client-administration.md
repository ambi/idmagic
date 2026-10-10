---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "同意の一覧と取り消し、セッション単位のトークン失効、管理用のクライアントの操作を、OAuth2 の公開契約の経由へ変えるだけである。同意とクライアントの不変条件、HTTP の応答、ドメインイベントは変えない。" }
---

# OAuth2 の同意、トークン失効、クライアント管理を公開契約にする

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 6 件は、ほかのモジュールから OAuth2 の非公開パッケージへの依存である。

| 利用側 | 参照先 | 使っているもの |
| --- | --- | --- |
| Authentication の `deps_http` と handler、IdManagement の `deps_http` と handler | `oauth2/consent/usecases` | 同意の一覧と取り消し（`ListConsentsForSub`、`RevokeConsent`、`ConsentDeps`） |
| Authentication の `session/handlers_http` | `oauth2/token/usecases` | ログアウトでのセッション単位のトークン失効（`RevokeTokensBySid`） |
| Application の handler | `oauth2/client/usecases` | 管理用のクライアントの作成、更新、秘密の発行と失効 |

## 対象範囲

- 同意の一覧と取り消し、管理用のクライアントの操作を、OAuth2 の公開パッケージとして公開する。
- ログアウトでのトークン失効は、Authentication が要求するポートとして定義し、組み立て地点で OAuth2 の実装を結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- 同意、トークン、クライアントの規則の変更。
- Application がプロトコル設定へ直接書き込む `table-write`。[所有者の書き込み](wi-88544-let-table-owners-publish-transactional-writes.md)で扱う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D2、D3：同意とクライアントの不変条件（秘密の上限、失効の伝播）は OAuth2 に残し、操作を公開する。
- D4：ログアウトのトークン失効は、Authentication から OAuth2 への辺であり、OAuth2 から Authentication への辺と循環する。要求する側の Authentication がポートを定義し、OAuth2 が実装する形を第一候補にする。

| 案 | 判断 |
| --- | --- |
| 同意とクライアントは公開操作、失効は要求側のポート | 採る |
| 同意の保存先を公開する | 採らない。取り消しの伝播（トークンの失効）を迂回できる |

## タスク

- [ ] T001 [Design] 公開する操作とポートの型を決める。
- [ ] T002 [App] 利用側と組み立て地点を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- ログアウトの失効のポートを組み立て地点で結び忘れると、ログアウト後もトークンが有効なまま残る。
  構築関数の必須引数にして、渡し忘れをコンパイルで拒否する。
