---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-27
priority: p3
group: federation-protocols
change_kind: feature
affected_spec:
  - { path: docs/modules/application/features/catalog.md, requirement: REQ-APPLICATION-007 }
---

# よく使う SaaS の統合テンプレートから Application を作成できるようにする

## 動機

Okta Integration Network と Entra ID のアプリギャラリーは、よく使う SaaS の事前定義のテンプレートを多数持つ。
管理者はプロトコルの設定を一から組まず、テンプレートを選ぶだけで、推奨のプロトコルのバインディング、デフォルトのクレームの開示、アイコン、必要な入力（ACS URL の欄、entityID の形など）の揃った Application を作れる。
IdMagic では Application、OIDC、SAML、WS-Fed のバインディング、下流への SCIM のプロビジョニングを手で作れるが、毎回バインディングとクレームを一から設定するのは煩雑で誤りやすい。

## 対象範囲

- リポジトリに同梱する、テナントに依存しない読み取り専用のテンプレートのカタログを持つ。テンプレートは、推奨のバインディングの種類、デフォルトのクレームの開示、アイコン、必要な入力の定義、必要ならプロビジョニングのデフォルトを持つ。
- テンプレートと管理者の入力から、Application とバインディングとクレームの開示を一度に作る操作を加える。作成の後は通常の Application と同じに扱う。
- 管理コンソールに、テンプレートの一覧と、テンプレートからの作成の画面を加える。

## 対象外

- 外部のレジストリ（OIN、Entra のギャラリー）からのオンラインの取り込み。
- テナントが独自のテンプレートを登録すること。
- テンプレートの更新を作成済みの Application へ反映すること。

## 設計

テンプレートのカタログを Application に置くか ApplicationCatalog に置くかを着手時に決める。
ApplicationCatalog は Application と各プロトコルの管理の編成を持つので、作成の操作はそこへ置く形が第一の候補である。
テンプレートの入力の定義は、各プロトコルの既存の検証を通す形にし、テンプレートの側に同じ検証を書かない。

## 検証

- テンプレートから作った Application が、手で作った同じ設定の Application と同じバインディングとクレームを持つことを観測する受け入れのテスト。
- 同梱のテンプレートが各プロトコルの検証を通ることを、テンプレートごとに確かめるテスト。
- `mise run verify-ui`
- `mise run verify`

## リスク

テンプレートの既定値が SaaS の側の変更で古くなると、作成した Application が動かない。
テンプレートに確認した日付と出典を持たせる。
