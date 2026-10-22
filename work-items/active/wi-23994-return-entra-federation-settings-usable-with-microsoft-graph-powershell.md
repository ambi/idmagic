---
status: in_progress
authors: [tn]
risk: medium
created_at: 2026-10-23
priority: p3
depends_on: [wi-97369-issue-entra-federation-tokens-for-the-realm-and-issuer-entra-expects]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
primary_use_cases:
  - id: graph-settings-use-the-signing-certificate
    requirement: REQ-WSFEDERATION-001
    observable_result: 構成結果の証明書がメタデータと一致し、Entra へ発行した Assertion を検証でき、Graph の名前と値を持つ作成と更新のコマンドを返す。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_graph_examples_test.go, name: TestConfigureEntraFederationReturnsUsableGraphSettings, task: test-go }
    fault_model: 証明書の説明文や検証用の旧証明書を返す、廃止された名前を使う、サインアウトやプロトコルや MFA の値を落とす。
  - id: copy-graph-commands-and-localize-limitations
    requirement: REQ-WSFEDERATION-001
    observable_result: 管理者が表示した作成と更新のコマンドをコピーし、制限を日本語と英語で読める。
    boundary: adapter
    test: { path: frontend/src/features/admin-entra-federation/AdminEntraFederationPage.test.tsx, name: Graph のコマンドをコピーし制限を翻訳する, task: test-ui-unit }
    fault_model: コピーが別のコマンドを渡す、制限コードや内部の英語の文を直接表示する。
---

# Entra の構成の結果を、Microsoft Graph PowerShell でそのまま使える形で返す

## 動機

Entra の構成の操作は、管理者が Entra 側に設定する値を `powershell` として返し、管理画面はそれを名前と値の組で表示する。
その値は、今の Entra の設定にそのまま使えない。

| 項目 | Microsoft Graph の設定（`internalDomainFederation`、`New-MgDomainFederationConfiguration`） | 現在の応答 |
| --- | --- | --- |
| 項目の名前 | `IssuerUri`、`PassiveSignInUri`、`ActiveSignInUri`、`MetadataExchangeUri`、`SignOutUri` | 廃止された MSOnline の名前（`PassiveLogOnUri`、`ActiveLogOnUri`） |
| 署名の証明書 | `SigningCertificate` に、証明書の公開部分の base64 | 「メタデータの証明書を使え」という英語の文 |
| 受動的なプロトコル | `PreferredAuthenticationProtocol` を `wsFed` と明示しなければ受動的なフローが動かない | 返さない |
| サインアウト | `SignOutUri` | 返さない。Entra からのサインアウトで idmagic のセッションが残る |
| MFA | `FederatedIdpMfaBehavior` | 返さない |

一次資料：[internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation)、[SAML 2.0 の IdP の構成例](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-fed-saml-idp)。

## 対象範囲

- 構成の結果に、Graph の名前の設定値（`SigningCertificate` は現在の有効な証明書の base64、次の鍵があれば `NextSigningCertificate`）を返す。
- 値を埋めた `New-MgDomainFederationConfiguration` と `Update-MgDomainFederationConfiguration` のコマンドを返し、管理画面でコピーできるようにする。
- `SignOutUri` に、`wsignout1.0` を受ける受動的なエンドポイントを渡す。
- `known_limitations` などの英語の文を、`code` と辞書の文言に改める。

## 対象外

- idmagic から Microsoft Graph を呼んで Entra を自動で構成すること。

## 設計

コマンドの組み立ては、構成の値からの決定論的な計算にし、PowerShell の文字列のエスケープを単体テストで確かめる。
`FederatedIdpMfaBehavior` の既定の案内は、wi-84807 の実装の有無で変わるので、wi-84807 より先に実装するときは `acceptIfMfaDoneByFederatedIdp` を案内する。

### 要件の差分

REQ-WSFEDERATION-001 に次を追加する。

- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、Microsoft Graph PowerShell の名前を持つ設定値、有効な XML 署名証明書の公開部分の base64、作成と更新のコマンドを返す。
- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、`SignOutUri` に受動的なエンドポイントを、`PreferredAuthenticationProtocol` に `wsFed` を、`FederatedIdpMfaBehavior` に `acceptIfMfaDoneByFederatedIdp` を返す。
- 管理者が構成の結果を表示したとき、WsFederation は、作成と更新のコマンドをそれぞれコピーできる形で表示し、制限のコードを選択した言語の文言で表示する。

### 採用する方式

2026-10-23 に Microsoft の一次資料で確認した。
[internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation) は証明書の公開部分の base64、`wsFed` の明示、MFA の値を定める。
[New-MgDomainFederationConfiguration](https://learn.microsoft.com/en-us/powershell/module/microsoft.graph.identity.directorymanagement/new-mgdomainfederationconfiguration) と [Update-MgDomainFederationConfiguration](https://learn.microsoft.com/en-us/powershell/module/microsoft.graph.identity.directorymanagement/update-mgdomainfederationconfiguration) のパラメーター名を使う。
更新の ID は同じドメインの `Get-MgDomainFederationConfiguration` から取得し、プレースホルダーを残さない。

証明書は既存の `FederationSigner.Resolve(ctx).Certificate().Raw` から取得する。
検証用に残した旧証明書を次の証明書と見なさない。
現在の鍵モデルは事前公開の状態を持たないため、次の証明書の取得と `NextSigningCertificate` の追加は wi-50764 が状態の追加とともに行う。
署名器の取得は RP の保存より前に行い、取得に失敗した要求では保存と成功イベントを残さない。
プロバイダー障害の HTTP への変換は既存の共通の内部エラー処理を使う。

純粋な計算は HTTP アダプター内の次の操作に閉じ、既存のモジュール境界と依存を保つ。

```go
func entraPowerShellCommands(domain string, settings map[string]string) powerShellCommands
func powerShellLiteral(value string) string
```

設定名は製品が固定し、値だけを単一引用符のリテラルとして渡す。
値に含まれる単一引用符を二重化し、変数展開、改行、バッククォートによるコードへの脱出を防ぐ。
コマンドの順序はキーのソートで安定させる。
コマンドの整形は仕様の細部とし（分類 b）、値の意味とリテラルの保存を検査する。
Microsoft Graph の自動呼び出し、Graph の権限の取得、実テナントでの実行は対象外とする。

### 計画

1. TypeSpec の応答に `commands` とコードを持つ制限を定め、要件を先に更新する。
2. HTTP の構成結果の証明書をメタデータと Assertion の検証で照合し、コマンドのリテラルを単体テストとファジングで検査する。
3. 辞書の文言とコピーの境界を UI 単体テストで検査し、正式な追加画面をブラウザーで確認する。

## タスク

- [x] T001 [Spec] REQ-WSFEDERATION-001 と `ConfigureEntraFederation` の応答を改める。
- [x] T002 [Acceptance] `TestConfigureEntraFederationReturnsUsableGraphSettings` で、Graph の設定、メタデータ、署名した Assertion を照合する。
- [x] T003 [App] 応答とコマンドの組み立てを実装し、`TestEntraPowerShellCommandsPreserveLiteralValues` と `FuzzPowerShellLiteral` で引用を検査する。
- [x] T004 [UI] 「Graph のコマンドをコピーし制限を翻訳する」でコピーと辞書の文言を検査し、`entra-source-anchor.spec.ts` で実際の構成結果を表示する。
- [x] T005 [Verify] 検証欄の検査を行い、七つの視点で差分をレビューする。

## 検証

- `mise run test-go-test -- ./backend/wsfederation/internal/handlers_http <test>`：受け入れの RED/GREEN と純粋な計算。
- `mise run test-go-fuzz -- ./backend/wsfederation/internal/handlers_http FuzzPowerShellLiteral 5s`：文字列の引用。
- `mise run test-go-changed -- 96d879f2a`：チェックポイント後も実装前のコミットから変更したハンドラーと逆依存を選ぶ race 検査。
- `mise run test-ui-unit-file -- src/features/admin-entra-federation/AdminEntraFederationPage.test.tsx` と `mise run verify-ui`：コピー、翻訳、型とビルド。
- `mise run test-ui-e2e-file -- tests/e2e/entra-source-anchor.spec.ts`：正式な画面からの構成結果。
- `mise run test-go-mutation -- backend/wsfederation/internal/handlers_http 2 off /tmp/idmagic-wi-23994-mutation.json 96d879f2a`：変更行の検出能力。
- `mise run check`、`mise run check-contract-drift`、`mise run check-spec-impact`、`mise run render-docs`：仕様、契約、生成表示。
- 実テナントでコマンドが通ることは wi-79 で確かめる。

## リスク

- 証明書の値を誤ると、Entra は署名を検証できずサインインを拒否する。メタデータの証明書と同じ値であることをテストで確かめる。

## 要約

`spec-diff` が示す変更は REQ-WSFEDERATION-001 と `ConfigureEntraFederation` である。
構成の応答を Microsoft Graph PowerShell の設定名と有効な XML 署名証明書の base64 に改め、サインアウト先、`wsFed`、`acceptIfMfaDoneByFederatedIdp` と、値を埋めた作成と更新のコマンドを返す。
管理画面は二つのコマンドをコピーでき、制限はコードから辞書の文言へ対応付ける。
PowerShell の引用は通常と曲がった単一引用符を二重化し、純粋な計算のテストとファジングで文字列を保存する性質を検査する。

構文変異に生存と未被覆はなく、コンパイルできない 14 件は文字列連結への算術変異である。
証明書を説明文に取り違える故障は構文変異で表せないため、手動の注入で検査する。
実テナントでのコマンドの実行は wi-79、事前公開する次の鍵と `NextSigningCertificate` の接続は wi-50764、MFA の要求への対応は wi-84807 に残す。
wi-50764 はこの応答を使って次の証明書を登録するため、この項目への依存を追加した。
