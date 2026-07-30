---
status: suggested
authors: [tn]
created_at: 2026-07-31
---

# ADR-149: `IdentityProviderConnection` の状態を `Active`/`Disabled` の2値に単純化する

## コンテキスト

[[wi-309-external-identity-provider-admin-ui-consistency]] の実機検証で、
`IdentityProviderConnectionStatus` の `Draft` が「まだ全項目が揃っていない書きかけの接続」を
表す状態として導入されたにもかかわらず、`Save()` が新規作成時点で `Validate()`
(全必須フィールドの充足チェック) を要求するため、DB に存在する接続は
Draft/Active/Disabled のいずれであっても常に「保存可能な形」を満たしていることが判明した。
`Draft` と `Disabled` はどちらも「login routing に使われない」という点で観測可能な違いがなく、
実質的に同じ状態が2つの名前を持っているだけになっている。

一方でこの状態は次の2箇所で実害を生んでいた。

- `deleteAdmin` は `status != Disabled` なら 409 で拒否するため
  (`backend/authentication/federation/handlers_http/routes.go:294-296`)、一度も有効化していない
  Draft を消すには「有効化 → 無効化 → 削除」という実質何もしていない3手順が必須になる。
- `updateAdmin` は編集の種類を問わず無条件に `status` を `Draft` に戻すため
  (`routes.go:274`)、表示名を直すだけの軽微な編集でも稼働中の接続が黙って `Active` から
  外れ、実ユーザーのログインが止まる副作用がある (この副作用自体への対処は [[ADR-150]] ではなく
  本 WI のユースケース層改修で扱う)。

## 決定

`IdentityProviderConnectionStatus` を `Active` / `Disabled` の2値のみに単純化する。

- 作成直後の初期状態は `Disabled` (安全側のデフォルト。従来 `Draft` と呼んでいた状態を吸収する)。
- 許可される遷移は `Active → Disabled` と `Disabled → Active` のみ。
- 削除は `Active` / `Disabled` どちらからでも可能にする (バックエンドの状態ガードを撤去)。
  `Active` な接続を削除する際の確認 UX はフロントエンド側の責務とする。

**既存データの移行**: `infra/schema/postgres.sql` の `identity_provider_connections.status`
CHECK 制約から `'draft'` を除去する schema 変更とは別に (ADR-071 §5: データ移行は宣言的
schema に混ぜない)、既存の `status = 'draft'` 行を `'disabled'` へ一方向に更新する
one-off script を
`infra/schema/data-migrations/2026-07-31-identity-provider-connections-draft-to-disabled.sql`
に置く。CHECK 制約の更新を適用する前に一度この script を実行する。`Active` への昇格は
必ず管理者の明示操作 (activate 呼び出し) を経由させ、この script 自体が `Active` に
昇格させることはない (誤って意図せず `Active` にしてログインを許可してしまう事故を避ける
ため)。

## 却下した代替案

- **`Draft` を維持し `Draft→Disabled` 遷移だけ追加する**: 遷移表は直るが、「保存可能=Validate
  済み」以上の情報を持たない状態がもう1つ増えるだけで、`Draft` と `Disabled` の意味的な
  重複という根本原因は残る。
- **`lifecycle_workflows` (`draft/enabled/disabled/archived` の4状態) に合わせて4状態化する**:
  `lifecycle_workflows` の `draft` は「本当にまだ書きかけ」を表せる (プロセス定義の途中保存が
  可能) が、`IdentityProviderConnection` は `Save()` 時点で必須項目が全て揃っていなければ
  ならないため、同じ設計を輸入しても `draft` が「書きかけ」を表す実体を持たない。

## 影響

- SCL (`spec/contexts/authentication.yaml`):
  - `models.IdentityProviderConnectionStatus` から `Draft` を削除。
  - `states.IdentityProviderConnectionLifecycle` を `Active ⇄ Disabled` の2状態遷移に再定義。
  - `interfaces.CreateIdentityProviderConnection` の description を「作成直後は Disabled」に変更。
  - `interfaces.DeleteIdentityProviderConnection` の description を「いつでも削除できる」に変更。
- Go: `backend/authentication/federation/domain` の `ConnectionStatus` 定数と
  `Activate`/`Disable`、`handlers_http` の `createAdmin` 初期状態と `deleteAdmin` の
  ステータスガード撤去。
- Data: `infra/schema/postgres.sql` の CHECK 制約更新と `draft → disabled` 一方向移行 SQL。
