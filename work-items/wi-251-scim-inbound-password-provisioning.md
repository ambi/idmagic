---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Scim:
      - standards.RFC7643.RFC7643-CORE-RESOURCES
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
  source:
    - backend/scim/domain/mutation.go
    - backend/scim/usecases/users.go
  tests:
    - backend/scim/domain/mutation_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7643, requirement: RFC7643-CORE-RESOURCES }
  - { context: Scim, kind: interface, element: CreateScimUser }
  - { context: Scim, kind: interface, element: UpdateScimUser }
---

# SCIM inbound password provisioning への対応可否を判断し、対応する場合は実装する

## Motivation

SCIM core User schema の `password` 属性(write-only)により、外部 IdP が
ユーザーの初期パスワードやパスワードリセット値を push できる。現状
CreateScimUser/UpdateScimUser/PatchScimUser はこの属性を黙殺する。

## Scope

**実装に入る前に、対応するかどうかの方針決定を最優先で行う**(Risk Notes 参照)。
対応する場合のみ、以下を実装する。

- `password` 属性の受け入れとハッシュ化・格納方針(既存の Argon2id パスワード
  格納経路との整合)を定義する。
- 既存のパスワードポリシー([[wi-92-configurable-password-policy]] があれば参照)
  との整合を確認し、ポリシー違反の password は拒否する。
- password 変更が内部の認証状態(session 無効化、MFA 再認証要求等)に与える
  影響を明示する。

## Out of Scope

- SCIM 経由でのパスワード**読み取り**(そもそも write-only 属性であり RFC でも禁止)。
- 対応しないという判断になった場合、`password` 属性は今後も silently ignore
  し続ける(その場合も `RFC7643-CORE-RESOURCES` の `reason` に明記する)。

## Plan

- 対応可否の判断を ADR として先に記録する(却下する場合もその理由を残す)。
- 対応する場合、平文 password が SCIM request body・ログ・トレースに残らない
  ことを実装・テストの両方で保証する。

## Tasks

- [ ] T000 [Decision] 対応するかどうかを ADR で判断する。却下する場合は
      `spec/contexts/scim.yaml` の `RFC7643-CORE-RESOURCES.reason` に明記して
      work item を `cancelled` にする。
- [ ] T001 [SCL] (対応する場合) password の契約を `spec/contexts/scim.yaml` に追加する。
- [ ] T002 [Domain] RED: password のハッシュ化・ポリシー検証 test を先に失敗させて実装する。
- [ ] T003 [Usecase/Adapter] RED: CreateScimUser/UpdateScimUser/PatchScimUser の
      password 処理 HTTP contract test を先に失敗させて実装する。ログ・エラー
      メッセージに平文 password が含まれないことを検証する test を含める。
- [ ] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: password 付きで User を作成し、ハッシュ化されて格納されること(平文が
  どこにも残らないこと)を確認する。
- 手動: パスワードポリシーに違反する password が拒否されることを確認する。

## Risk Notes

**セキュリティ上の判断が必要**: 第三者 SCIM client からの平文パスワード受け入れは
攻撃面を広げる(弱いパスワードの一括投入、パスワードポリシー回避、transport/ログ
上の露出リスク)。既存の自己サービスパスワードフロー・パスワードポリシーとの
整合が取れない場合は、恒久的に非対応とする判断も正当である。実装前に必ず
方針を ADR で確定してから着手する(T000 を他の Task より先に完了させる)。
