---
status: in_progress
authors: [tn]
risk: high
created_at: 2026-10-23
priority: p2
depends_on: [wi-23994-return-entra-federation-settings-usable-with-microsoft-graph-powershell]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/signing-keys/features/separation.md, requirement: REQ-SIGNINGKEYS-006 }
  - { path: docs/modules/signing-keys/features/separation.md, requirement: REQ-SIGNINGKEYS-007 }
  - { path: docs/modules/signing-keys/features/lifecycle.md, requirement: REQ-SIGNINGKEYS-001 }
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/modules/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.RotateTenantSigningKey }
  - { path: spec/modules/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.ListAdminKeys }
  - { path: spec/modules/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.GetAdminKey }
  - { path: spec/modules/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.DisableTenantKey }
  - { path: spec/modules/signing-keys/models.tsp, symbol: IdMagic.SigningKeys.AdminKeyResponse }
  - { path: spec/modules/signing-keys/models.tsp, symbol: IdMagic.SigningKeys.AdminRotateKeyRequest }
  - { path: spec/modules/signing-keys/models.tsp, symbol: IdMagic.SigningKeys.SigningKeyPrepared }
  - { path: spec/modules/signing-keys/models.tsp, symbol: IdMagic.SigningKeys.SigningKeyRotated }
  - { path: spec/modules/signing-keys/models.tsp, symbol: IdMagic.SigningKeys.SigningKeyState }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
primary_use_cases:
  - id: publish-before-switching-the-entra-signature
    requirement: REQ-SIGNINGKEYS-006
    observable_result: 準備中の Assertion は旧証明書で検証でき、確認した切り替えの後だけ新証明書で検証できる。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_rotation_examples_test.go, name: TestEntraRotationPublishesBeforeSwitchingTheSignature, task: test-go }
    fault_model: 準備で署名鍵を切り替える、古い証明書を次の証明書とする、確認なしに切り替える。
  - id: preserve-vault-selection-across-restart
    requirement: REQ-SIGNINGKEYS-007
    observable_result: 再起動と別のストアの要求でも、有効なバージョンと準備済みの証明書を保つ。
    boundary: adapter
    test: { path: backend/signingkeys/internal/keys_vault/staged_test.go, name: TestVaultPreparedSelectionSurvivesRestart, task: test-go }
    fault_model: Vault の最新バージョンを無条件に署名へ使う、状態ストアの選択を読み直さない。
  - id: compose-vault-with-durable-selection
    requirement: REQ-SIGNINGKEYS-007
    observable_result: PostgreSQL と Vault を指定した実際の起動経路で、準備後に再構築しても旧鍵を Vault 経由で使い続ける。
    boundary: adapter
    test: { path: backend/cmd/internal/bootstrap/vault_rotation_test.go, name: TestAssembleKeepsTheVaultXMLSelectionInPostgres, task: test-go }
    fault_model: Vault を通常の DB 鍵へ取り違える、永続状態の配線をメモリへ取り違える。
---

# XML フェデレーションの鍵のローテーションで、Entra のサインインを止めない

## 動機

`XmlFederationSigning` の鍵のローテーションは、新しい鍵で直ちに署名を始め、古い鍵を 7 日の間メタデータに残す（REQ-SIGNINGKEYS-006）。
これは、メタデータを読み直す RP が古い鍵で署名された Assertion を検証し続けるための重なりである。
Entra は逆の向きの重なりを要する。

- Entra は、ドメインに設定した `signingCertificate` と `nextSigningCertificate` で Assertion を検証する（[internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation)）。
- Entra がフェデレーションのメタデータから新しい証明書を取り込むのは、現在の証明書の期限の 30 日前からである。
- idmagic の証明書の期限は 30 年（`keys_jose/certificate.go`）なので、自動の更新は実際には始まらない。

そのため、管理者が `XmlFederationSigning` の鍵をローテーションすると、その時点から Entra が知らない鍵で署名することになり、フェデレーションしたドメインの Microsoft 365 のサインインが止まる。
管理者が Entra の設定を更新するまで復旧しない。
管理画面とローテーションの API は、この影響を知らせない。
定期のバッチは `Signing` の鍵だけをローテーションするので、今は管理者の操作のときだけ起きる。

## 対象範囲

- 次の鍵を、署名に使う前にメタデータで公開する段階を設け、WS-* と既定 SAML が共有する XML 鍵は必ずその段階を経て切り替える。
- Entra の構成の結果で、次の鍵の証明書を `nextSigningCertificate` として渡せるようにする。
- REQ-SIGNINGKEYS-006 と鍵のライフサイクルの状態遷移を改める。

## 対象外

- SAML の SP ごとの鍵の扱いの見直し。SAML のプロファイルの鍵は、この段階を任意で使えるようにするだけにする。
- 連携先への証明書の自動登録、自動の登録完了の確認、実際の Entra テナントでの疎通、XML 鍵の定期バッチ。

## 設計

| 案 | 採否 | 理由 |
| --- | --- | --- |
| 次の鍵を先に公開し、管理者の確認の後に署名の鍵へ切り替える | 採用 | 全連携先への登録を確認してから切り替える。時間経過では登録を確認できない |
| 証明書の期限を短くし、Entra の自動更新に任せる | 採らない | Microsoft の資料間で取得開始の時期が 30 日と 35 日に分かれ、第三者 IdP のメタデータ取得の成功を保証できない。証明書の期限も全連携に影響する |
| ローテーションの API で警告を出すだけにする | 採らない | 停止を防げない |

Entra の [証明書の更新の説明](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-fed-o365-certs)と [Graph のフェデレーション構成](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation)から、時間経過に頼らず管理者が次の証明書の登録を確認する案を選んだ。

次の証明書を Entra に登録するため、先に wi-23994 で有効な証明書を含む Graph PowerShell の構成結果を整える。
事前公開の状態を追加するとき、この応答へ `NextSigningCertificate` を接続する。
Vault の提供元は再起動時に最新のバージョンを署名用に取り込むため、事前公開しただけの鍵を署名に使わない永続的な選択も、この設計で解決する。

### 要件と契約の差分

- REQ-SIGNINGKEYS-006：XML 鍵の `Prepared` を公開し、有効な署名鍵を保った後、管理者が登録済みと確認した kid だけを有効にする。準備の再実行は同じ kid を返し、切り替えの再実行は猶予を延長しない。準備した鍵は一つだけとする。
- REQ-SIGNINGKEYS-001：XML の二段階操作を追加する。通常の即時ローテーションは、二段階を要求する鍵集合では拒否する。署名鍵の切り替え時に旧鍵の 7 日の猶予を開始する。
- REQ-SIGNINGKEYS-007：PostgreSQL と Vault の組み合わせでも有効と準備済みの選択を再起動後に保つ。
- REQ-WSFEDERATION-001：構成結果に準備済み証明書の `NextSigningCertificate` を含める。準備済みでない検証用の旧証明書は含めない。
- `RotateTenantSigningKey`：既存の要求へ `action`（`rotate`、`prepare`、`activate`）、`kid`、`trust_updated` を追加する。activate は XML の用途、kid、`trust_updated: true` を要する。不正な組み合わせと即時切り替えの拒否は既存の 400 `invalid_request` を使う。
- `AdminKeyResponse`：用途、スコープ、準備済みか、証明書の公開部分の base64 を返す。秘密鍵を返さない。
- 鍵の一覧、取得、無効化は query の `usage` と `scope_id` で XML 鍵の集合も選択できる。不正な組み合わせは 400 `invalid_request` で拒否する。

### 境界と永続化

利用者の指摘に従い、未リリースの TO BE として共有する XML 鍵は常に二段階にする。
SigningKeys が状態と選択を所有する（D1）。WsFederation は既存の署名提供元の技術的な公開契約から準備済みの証明書を読む（D5）。
共有契約は Entra や RP の型を受け取らない。SAML の静的な署名器はローテーションを提供しないため、準備済み証明書を持たない。
構成の度に WsFederation を読み直して分岐する案は採らない。鍵ストア内の直列化と別になるため、バッチや競合するローテーションが制約を飛び越せる。
必須にする条件は XML の用途と default のスコープだけで決まり、Entra の RP の状態に依存しない。
SAML 専用プロファイルのスコープでは、即時ローテーションを維持し、任意で prepare と activate を使える。

Local は mutex の内側、Database は既存の鍵集合の advisory lock の内側で、準備、切り替え、無効化を直列化する。
`signing_keys` に `prepared` を追加する。
Vault は鍵の秘密部分を持ち出さず、公開鍵、証明書、バージョン、状態の写しだけを SigningKeys 所有の `vault_signing_key_sets` へ保存する。
PostgreSQL の組み立て地点は必須の状態ストアを Vault に渡し、要求ごとに保存した選択を読む。プロセス内のキャッシュを正しい選択の根拠にしない。
Vault のバージョン作成後に DB の保存が失敗しても、有効な選択を変えず、未参照のバージョンを署名へ使わない。
memory の構成は状態全体と Entra の RP が非永続であり、同じ範囲で公開鍵の状態も持つ。

```go
type RotationResult struct { Next, Previous *SigningKey; Changed bool }
func ActivatePrepared(keys []*SigningKey, kid string, now time.Time, grace time.Duration) (RotationResult, error)
type FederationKeyStore interface {
    Prepare(context.Context, time.Time) (RotationResult, error)
    Activate(context.Context, string, time.Time, time.Duration) (RotationResult, error)
}
```

準備と選択はポートの作用、状態遷移の判断は入力の鍵と時刻に対する計算に分ける。
未知の kid、退役済みや無効化済みの kid の activate は拒否し、有効な鍵と記録を変えない。
`SigningKeyRotated` は切り替えでだけ発行する。準備は `SigningKeyPrepared` を発行し、同じ準備の再実行では重複させない。

### 互換性と移行と復旧

公開状態は unpublished である。JWT の鍵、署名アルゴリズム、SAML 専用スコープの即時切り替えは維持する。
既存の DB 行の `prepared` の既定値は false とする。共有 XML 鍵の即時切り替えは提供元の内側で拒否するため、構成の移行や起動時の RP の走査を要しない。
Vault の状態がまだ保存されていない初回は、従来と同じ最新のバージョンを取り込み、それ以降は保存した選択を使う。
切り替え前は準備済みの鍵を無効化すれば取り消せる。切り替え後の逆方向の再有効化は提供せず、以前の証明書と新しい証明書の両方を Entra に残して信頼を復旧する。
相手の製品への設定を自動で検査する保証は追加しない。管理者は共有鍵を使う全連携先への登録を確認する。

### 検証の選択

基準は `e3fe207fe` とする。
Entra の構成、鍵の準備、メタデータの取得、Assertion の発行と署名の検証、確認後の切り替えを同じ HTTP スタックで観測する。
Local と PostgreSQL の準備と切り替えは共有契約で、Vault の再起動と複数のストアからの読み直しは状態ストアの境界で検査する。
状態マトリクスから純粋な遷移テストを実行し、テナント、用途、スコープ、拒否後の状態、重複、猶予の境界を含める。
`mise run check-spec`、局所の `test-go-test` と `test-go-package`、`sqlc-generate`、`check-schema`、`check-contract-drift`、`check-operation-references`、`lint-go`、`test-go-changed -- e3fe207fe`、`check` を選ぶ。
画面の実装は既存の応答表示を利用するため変えず、対象の `test-ui-e2e-file -- tests/e2e/entra-source-anchor.spec.ts` で次の証明書を含むコマンドを観測する。UI 本体を変えないため `verify-ui` は追加しない。
変更した Go パッケージには `test-go-mutation -- <package> 2 off <report> e3fe207fe` を使い、最新バージョンへの誤った切り替え、事前公開制約の未配線、証明書の取り違えは手で注入する。
最終的に `spec-diff`、`render-docs`、`generate-contract`、`check-published-api-compat` と記録の検査を行う。

## 計画

1. T001 で設計の候補を決める。
2. 状態遷移を変えるので、SigningKeyLifecycle の遷移の表とモデルテストを同時に改める。

## タスク

- [x] T001 [Plan] Entra の証明書の更新の手順を確かめ、設計を決める。共有 XML 鍵はすべて二段階とする。
- [x] T002 [Spec] REQ-SIGNINGKEYS-006、状態遷移、Entra の定型設定の運用の記述を改める。`check-spec` が成功。
- [x] T003 [Acceptance] `TestEntraRotationPublishesBeforeSwitchingTheSignature` が、準備を実装する前は 500 で失敗し、実装後は旧証明書での署名の維持と確認後の切り替えを検証して成功した（REQ-SIGNINGKEYS-006）。
- [x] T004 [App] 次の鍵の公開と切り替えを実装する。Local と PostgreSQL の共有契約、Vault の再起動、実際の Assemble の配線、SAML と WS-* の HTTP テストが成功。
- [x] T005 [Verify] 変更を検証する。状態マトリクス、共有契約、HTTP、起動時配線、関連する race テスト、対象の E2E、静的検査、スキーマ収束を検査した。

## 検証

- `test-go-changed -- e3fe207fe` は変更したパッケージと逆依存の race 検査を担当する。`lint-go`、`check`、`spec-diff -- e3fe207fe`、仕様と契約の再生成も行う。
- `check-schema` は今回の列とテーブルの追加に対する収束検査であり、通常の Go テストとは別に Colima 上で行った。
- `TestPreparedActivationMatchesTheStateMatrix` は規範の状態表から有効化の結果を読む。共有契約は準備の競合、テナントとスコープの分離、公開、無効化、旧鍵の期限、アーカイブ、再実行を読む。
- `FuzzRotationConfirmation` は正当な有効化を受理し、同じ入力から登録済みの確認を外すと拒否することを判定基準にする。
- 変更行を対象にした機械的変異では、状態表、鍵集合の選択、HTTP のイベントと猶予、Vault の公開、起動時配線の観測を補強した。最終結果は 147 件を検出し、5 件が生存し、2 件が時間超過した。
- 生存した 4 件は Local と DB の初回生成に渡す猶予の算術変更である。初回は旧鍵が存在しないため結果を変えない。残る 1 件はメモリ状態の JSON 解析エラーを無視する変更であり、その状態は成功した JSON エンコードからだけ作られるため、本番の入力から解析エラーへ到達しない。
- DB の Begin の成功を失敗と判定する 2 件は、単独でも時間超過した。未解放のトランザクションを伴う変異であり、テストによる検出とは数えない。DB の状態のコミットと失敗時の不変性は実 PostgreSQL で別に観測する。
- 手動注入した未確認の Vault 鍵の選択、永続化の配線の切断、次の Graph 証明書の取り違えは、それぞれ `TestVaultPreparedSelectionSurvivesRestart`、`TestAssembleKeepsTheVaultXMLSelectionInPostgres`、`TestEntraRotationPublishesBeforeSwitchingTheSignature` が検出する。注入を戻して関連する race テストを再実行する。

## リスク

- 状態を増やすと、JWT の鍵と XML の鍵で遷移が分かれる。用途ごとの違いを遷移の表に明記する。

## 要約

共有 XML 鍵の切り替えを、Entra の設定有無によらず「準備して公開、全連携先への登録を確認、有効化」の二段階とした（REQ-SIGNINGKEYS-001、006）。
準備中は署名鍵を保ち、確認した kid の有効化で旧鍵の 7 日の猶予を始める。再実行は鍵、イベント、猶予を増やさない。管理 API に操作と鍵集合の選択を加え、Entra の構成結果と Graph コマンドに準備済み証明書を渡す（REQ-WSFEDERATION-001）。

Local、Database、Vault を同じ遷移に揃え、Vault の公開状態と選択を PostgreSQL に保存する。実際の起動時の配線を含め、再起動後も Vault の最新バージョンを無条件に署名へ使わない（REQ-SIGNINGKEYS-007）。
JWT と SAML 専用スコープの即時切り替えは維持する。自動登録、自動の登録確認、実 Entra テナントでの疎通と定期バッチは対象外とした。

変更耐性の限界として、機械的変異には初回生成の猶予の変更 4 件と到達しないメモリの解析エラー 1 件が残り、DB のトランザクションを未解放にする 2 件は単独再実行でも時間超過した。時間超過は検出と数えず、永続化と配線は主要テストと意味上の故障注入で検査した。
