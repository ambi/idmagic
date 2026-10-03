---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-042 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-043 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-060 }
---

# User、Group、Agent の名前を一つの値オブジェクトで比較し、User を作るすべての経路で同じ検証と動的グループの評価を行う

## 動機

IdManagement では、同じ種類の値と操作が、Aggregate や経路ごとに異なる振る舞いをしている。

| 対象 | 今の挙動 |
| --- | --- |
| 名前の比較 | User のユーザー名は大文字と小文字を区別して一意性を判定する。Group と Agent の名前は `strings.EqualFold` で区別せずに判定する。グループのメンバーシップの CSV は、ユーザー名を区別せずに照合する |
| メールアドレスの重複 | フェデレーションの JIT とメールアドレスの変更は重複を拒否し、管理者による作成は拒否しない |
| 動的グループの評価 | 管理者による作成と CSV の適用は、作った User を動的グループの規則で評価する。JIT は評価しないので、JIT で作った User は次の再評価まで動的グループに所属しない |

主要な製品と規格は、ユーザー名を大文字と小文字を区別せずに比較する。

| 製品・規格 | ユーザー名 | グループ名 |
| --- | --- | --- |
| Okta | ログイン名は大文字と小文字を区別しない | 一意である |
| Microsoft Entra ID | UPN は大文字と小文字を区別しない | 表示名は一意でなくてよい |
| Google Workspace | メールアドレスを小文字にそろえる | グループのメールアドレスで識別する |
| Keycloak | 保存時に小文字へ変換する | 同じ親の下で一意である |
| SCIM（RFC 7643） | `userName` は `caseExact: false` | `displayName` は `caseExact: false` |

## 対象範囲

- User のユーザー名、Group の名前、Agent の名前を、一つの値オブジェクト（名前）で正規化し、比較する。
  - 前後の空白を除いて保存する。
  - 入力された表記のまま保存し、表示する。
  - 大文字と小文字を区別せずに比較する。
  - 同じテナントの削除されていない同じ種類の Aggregate の間で一意にする。
- メールアドレスの重複を、User を作るすべての経路（管理者による作成、CSV の適用、フェデレーションの JIT）とメールアドレスの変更で、大文字と小文字を区別せずに拒否する。
- User を作るすべての経路を共有の仕組みにまとめ、リソース上限、名前とメールアドレスの一意性、属性スキーマの検証と、動的グループの評価を一か所で行う。
- 値オブジェクトの定義を IdManagement の `README.md` のモデルの節へ移し、各機能の要件はその定義を参照する。
- PostgreSQL の一意性の制約を、大文字と小文字を区別しない比較に合わせる。

## 対象外

- 既存データの移行。
  製品は未リリースなので、スキーマの宣言を変えるだけにする。
- メールアドレスの照合での Unicode の正規化（NFKC など）。

## 設計

`strings.EqualFold` と `strings.ToLower` は Unicode の一部の文字で結果が異なる。
名前の比較キーは一つの関数だけで作り、アプリケーションの判定とデータベースの制約が同じキーを使うようにする。
データベースの制約は、比較キーを保存した列への一意索引か、`lower()` の式索引で表す。
どちらにするかは、Go の比較キーと PostgreSQL の `lower()` が一致する範囲を確かめて着手時に決める。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| Keycloak と同じく小文字へ変換して保存する | 利用者が入力した表記を表示できなくなる。Okta、Entra ID は表記を保つ |
| User だけ大文字と小文字を区別したまま残す | 同じ製品の中で名前の扱いが Aggregate ごとに異なり、CSV の照合とも食い違う |

## 計画

1. 名前とメールアドレスの値オブジェクトを定義し、要件を差分として書く。
2. User を作る経路を共有の仕組みにまとめる。
3. Group と Agent の名前の比較を値オブジェクトへ置き換える。
4. スキーマの一意性の制約を更新する。

## タスク

- [ ] T001 [Spec] 値オブジェクトと要件の差分を書く。
- [ ] T002 [Acceptance] 大文字と小文字だけが異なるユーザー名の作成が拒否されることの RED を確認する。
- [ ] T003 [App] 作成の経路を共有の仕組みにまとめ、値オブジェクトで比較する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run verify`

## リスク

- 作成の経路をまとめると、CSV の適用やフェデレーションの JIT の、ほかの差まで変わる。
  経路ごとの差を観点表で洗い出し、意図した差だけを要件として残す。
