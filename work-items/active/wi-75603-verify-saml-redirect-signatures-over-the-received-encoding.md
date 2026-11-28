---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: spec/modules/saml/main.tsp, symbol: IdMagic.Saml.Operations.SamlSingleSignOn1 }
  - { path: spec/modules/saml/main.tsp, symbol: IdMagic.Saml.Operations.SamlSingleLogout1 }
---

# SAML の HTTP-Redirect の署名を、受け取ったままの URL エンコードで検証する

## 動機

SAML IdP は、署名を求める SP の AuthnRequest と LogoutRequest を HTTP-Redirect で受け取ると、`backend/saml/domain/authnrequest.go` の `validateRedirectSignature` で署名を検証する。
この関数はクエリを `url.ParseQuery` で一度デコードし、`url.QueryEscape` で符号化し直した文字列に対して署名を検証する。
SAML 2.0 Bindings 3.4.4.1 は、URL エンコードが一意でないことを理由に、検証は受け取ったクエリの元の URL エンコード値で行わなければならないとする。

符号化し直した文字列は、`url.QueryEscape` と同じ規則で符号化する SP のときだけ元の文字列と一致する。
16 進を小文字で書く SP（.NET の `HttpUtility.UrlEncode` を使う実装）、空白を `%20` で書く SP、予約されていない文字まで符号化する SP の正しい署名は、検証に失敗するはずである。
テストの要求はどれも IdMagic の同じ符号化で作っているので、この食い違いを見つけられない。
外部の相手の記録で欠陥が見つかった取り組み（wi-94480 ほか）は、こちらが要求を受け取る側を洗い出していなかった。

## 対象範囲

- 生のクエリ文字列から `SAMLRequest`（または `SAMLResponse`）、`RelayState`、`SigAlg` の値を、デコードせず受け取ったままの部分文字列として取り出し、3.4.4.1 の順序で連結して検証する。
- 同じ名前の引数が二度現れるクエリと、署名の対象に入らない引数の扱いを一次資料の文面から決め、宣言する。
- 手元で起動できる SP から、署名した HTTP-Redirect の AuthnRequest と LogoutRequest を記録して再生する。符号化の規則が違う SP を二つ以上含める（候補は SimpleSAMLphp、Shibboleth SP、Sustainsys.Saml2）。
- `SAML2Bindings-RedirectPost` の記述に、署名の検証が受け取った符号化に対して行われることを書く。

## 対象外

- 上流の SAML IdP からの応答の署名。[[wi-53342-accept-upstream-saml-assertions-signed-without-a-response-signature]] が扱う。
- HTTP-POST の XML 署名の検証。符号化の再構成を伴わないので変えない。
- RSA 以外の署名アルゴリズムの追加。

## 設計

`net/url` の `URL.RawQuery` は受け取ったままの文字列を保つので、`&` で分け、`=` の前の名前だけをデコードして比べ、値は元の部分文字列のまま使う。
`ValidateRequestSignature` の入力は今も `rawQuery` であり、呼び出し元（`usecases/signin.go`、`usecases/logout.go`）の変更は要らない見込みである。
呼び出し元が `RawQuery` ではなく符号化し直した値を渡していないかは、着手時に HTTP の入口から確かめる。

### 記録の取り方

SimpleSAMLphp と Shibboleth SP はコンテナで、Sustainsys.Saml2 は `dotnet` のサンプルで起動できる。
SP に IdMagic の IdP メタデータの代わりに記録用の受け口を渡し、SP が送るリダイレクトの URL をそのまま保存する。
SP の署名鍵は記録用に作った鍵であり、記録には証明書だけを残す。
一度だけの取得なら `mise` タスクにせず、取得方法に手順を書く。

## 計画

1. 3.4.4.1 の文面から、小文字の 16 進と `%20` を使う要求の期待値を書き、RED を作る。
2. 実物の SP の記録を取り、再生が今の実装で失敗するかを確かめる。失敗しなければ、手書きの RED と記録のどちらが実物を表すかを記録の側で確かめる。
3. 受け取ったままの値で検証する形に直す。

## タスク

- [ ] T001 [Spec] `SAML2Bindings-RedirectPost` に検証の対象の文字列を書く。
- [ ] T002 [Recording] 符号化の規則が違う SP から、署名した要求を記録する。
- [ ] T003 [Domain] 受け取った符号化での検証の RED と GREEN、重複する引数の拒否を単体で検査する。
- [ ] T004 [Acceptance] 記録の再生で、AuthnRequest と LogoutRequest がどちらも受理されることを確かめる。
- [ ] T005 [Verify] 差分から選んだ検査を実行する。

## 検証

- 記録のクエリの一文字を、デコード後が同じになる別の符号化（`%2F` を `%2f`）に変えると、署名の検証が失敗する。
- `mise run test-go`、`mise run lint-go`。

## リスク

重複する引数を許すと、署名した値と処理する値が食い違う。
署名の対象の引数が二度現れる要求は拒否する。
