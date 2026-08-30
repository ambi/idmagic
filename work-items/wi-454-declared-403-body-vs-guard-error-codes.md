---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-30
change_kind: bugfix
priority: p2
depends_on: []
affected_spec:
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminUser }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.AccessDeniedError }
---

# 403 が名乗る本体を、guard が実際に返すエラーコードに合わせる

## Motivation

`wi-386` はステータスコード集合を閉じたが、403 の**本体**が実装と食い違っていることを、その過程で数え上げた。403 を宣言する 272 operation のうち 270 が `AccessDeniedError` (`urn:idmagic:error:access_denied`) だけを名乗る。しかし 403 を書く guard は 3 つあり、返すコードは 3 通りである。

- `support_http.VerifyBrowserRequest` は `invalid_origin` と `csrf_failed` を返す。ブラウザーから呼ばれる 130 の handler がこれを通る。
- `support_http.WriteAccessTokenError` は `insufficient_scope` を返す。API アクセストークンで到達しうる管理 API のほぼ全部がこれを通るが、`InsufficientScopeError` を宣言しているのは 2 operation だけである。
- `Authenticator.WriteAdminAccessError` は `access_denied` を返す。宣言が合っているのはこれだけである。

`docs/api-rules.md` は「個々のエラーは `model <Name>Error is ProblemDetails;` と書き、どの `type` URN 接尾辞に対応するかを `@doc` で名指しする」と定めている。同じ 403 に 3 つのコードが乗っているのに 1 つしか宣言していないので、`type` を鍵に翻訳する UI は、実際に来る 2 つを辞書で引けない。

`wi-386` はこれを Out of Scope に置いた。本体の形は `wi-382` と `wi-385` の担当で、132 operation のうち 2 つだけを直せば残りとの不一致を新しく作ることになるためである。

## Scope

- `invalid_origin` / `csrf_failed` / `insufficient_scope` の本体モデルを宣言する。
- 各 operation の `<Op>Error403Body` を、その operation の手前に立つ guard が実際に返しうるコードの union にする。
- `check-security-controls` の R4 は 403 の本体型で scenario と突き合わせている。新しい型を足すと R4 が「その拒否を宣言する scenario が無い」と言うので、CSRF と origin の拒否を規範的シナリオとして書き、その id を検査から名指しする。`docs/development/specification-first-workflow.md` の「拒否のテスト」に従い、拒否が何を残さなかったかまで見る。
- 本体の一致を機械で見る手段を検討する。`wi-385` の `check-contract-drift` は成功応答の本体しか見ておらず、エラー本体は誰も突き合わせていない。

## Out of Scope

- ステータスコード集合。`wi-386` が閉じた。

## Verification

- `mise run check-spec`
- `mise run check-security-controls`
- `mise run verify`

## Risk Notes

`<Op>Error403Body` の union にメンバーを足すのは、生成クライアントにとっては受け取りうる型が増えることであり、既存の分岐は壊れない。ただし R4 が要求する scenario を書かずに型だけ足すと `mise run check` が落ちるので、シナリオと検査を同じ変更に含める。
