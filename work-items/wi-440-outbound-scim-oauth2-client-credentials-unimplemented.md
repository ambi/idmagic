---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-29
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-AUTHENTICATION }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.ProvisioningAuthMethod }
---

# 接続が受け付ける `oauth2_client_credentials` を、実際にトークンを取って使う

## Motivation

`ProvisioningAuthMethod` は `bearer_token` と `oauth2_client_credentials` の 2 値を受け付け、接続の登録画面も API もこの 2 つを提示する。ところが送出クライアントを組み立てる経路は認証方式を一切見ておらず、保存した資格情報をそのまま `Authorization: Bearer` の値として送る。

`oauth2_client_credentials` を選んだ接続では、これはクライアントシークレットをベアラートークンとして下流へ提示することを意味する。**下流が正しく実装されていれば必ず 401 になり、正しく実装されていなければシークレットがアクセストークンとして通ってしまう。** どちらも望ましくない。

さらに、401 は `RFC7644-OUT-ERROR-RESPONSE` の「再試行しない失敗」に落ちるので、管理者が見るのは配信の `dead_letter` だけで、「この認証方式は動かない」という事実はどこにも現れない。設定できるのに動かない選択肢が、失敗の原因として自分を名乗らないまま残っている。

[docs/contexts/provisioning/standards.md](../docs/contexts/provisioning/standards.md) の `RFC7644-OUT-AUTHENTICATION` は、この食い違いを避けるために「認証は `Authorization: Bearer` の 1 方式に限り、資格情報を得るための別の要求は送らない」という真の宣言を置いている。この work item が直せば、その行は方式ごとの分岐を書けるようになる。

## Scope

- クライアント資格情報フロー (RFC 6749 §4.4) でアクセストークンを取得し、下流への要求に使う。
- トークンエンドポイント、スコープ、クライアント認証方式を接続の設定として持つかどうかを決める。いまの資格情報モデルには `client_id` と `client_secret` に相当する欄しか無い。
- 取得したトークンの有効期限内での再利用と、期限切れおよび 401 での再取得を決める。
- 実装できないと決めるなら、逆に `ProvisioningAuthMethod` から値を取り除く。**受け付ける値と動く値を一致させることが目的であり、方向はどちらでもよい。**
- `RFC7644-OUT-AUTHENTICATION` の `Statement` と証拠テストを更新する。

## Out of Scope

- 相互 TLS、署名付き JWT クライアント認証などの他のクライアント認証方式。
- 下流ごとの独自の認証方式。
- 資格情報の保存方式そのもの。既存のエンベロープ暗号化を使う。

## Design

未定。着手時に、値を実装するのか取り除くのかを先に確定して本節に記録する。取り除く側を選ぶ場合、既存の接続が持ちうる値の移行が要るかを併せて調べる。

## Plan

1. 実装するか取り除くかを決める。
2. 選んだ側の RED を置く。
3. 実装し、規範行を更新する。

## Tasks

- [ ] T001 [Design] 実装と削除のどちらを取るかを確定する。
- [ ] T002 [Test] 選んだ側の RED を置く。
- [ ] T003 [App] 実装する。
- [ ] T004 [Spec] `RFC7644-OUT-AUTHENTICATION` と TypeSpec を更新する。
- [ ] T005 [Verify] `mise run check-spec`、`mise run verify`。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run test-go`
- `mise run verify`

## Risk Notes

リスクは medium。資格情報の扱いに触れるため、取得したトークンをログや監査記録へ漏らさないことを確かめる必要がある。列挙値を取り除く側を選ぶと公開 API の破壊的変更になるので、`mise run check-api-compat` の結果を根拠として残す。
