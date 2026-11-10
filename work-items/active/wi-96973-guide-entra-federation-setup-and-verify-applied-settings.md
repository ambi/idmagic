---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-11
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.EntraFederationProfile }
---

# Entra フェデレーションの導入を、設定の保存から反映確認まで案内する

## 動機

現在の Entra 管理画面は、IdMagic に RP を保存して Graph PowerShell のコマンドを表示する。
Entra へ反映していなくても一覧には「フェデレーション済み」と表示され、管理者が利用可能な状態と誤認する。
生成結果は画面内の一時状態にしかなく、再表示には同じ設定の再保存が必要になる。
Okta の[構成手順](https://help.okta.com/oie/en-us/content/topics/apps/office365-deployment/configure-sso.htm)も参考に、手動反映を採用したまま、準備、適用、確認を一つの操作の流れにする。

## 対象範囲

- ドメインの読み取り専用の詳細と専用の編集画面を用意し、保存済み設定、期待する Entra 設定、公開証明書、適用用コマンドを変更操作なしで再取得する。
- Graph SDK の準備、必要な読み取り権限と変更権限、接続先テナントの確認、変更前の設定の保存、差分、適用、読み戻しを PowerShell の手順として提供する。
- ローカル保存と Entra 登録値の照合結果を分け、未確認、一致、不一致、再確認が必要な状態と、確認対象と時刻を表示する。
- AD 認証委譲、UPN と ImmutableID、Connect のコンピューター同期、SCP、SYSTEM の通信を導入の前提として案内し、sync join の同期待ち、端末参加、PRT、アプリ SSO を別の確認項目にする。

## 対象外

- Graph の自動書き込み、Graph 資格情報の IdMagic への保存、Entra Connect の同梱。
- 実 Windows の状態を管理画面が自動取得する機能。実機の受入試験は [[wi-79-entra-real-tenant-end-to-end-verification]] が扱う。
- ドメインの解除と XML 鍵の更新操作。それぞれ [[wi-95114-remove-entra-federation-one-domain-at-a-time]]、[[wi-87286-guide-xml-federation-signing-key-rollover]] が扱う。

## 設計

### 反映を確認する境界

Graph PowerShell が取得した登録値を JSON で取り込み、WsFederation が現在の期待値と比較する手動方式を採る。
比較対象はテナント、ドメイン、IssuerUri、各 URL、認証プロトコル、MFA の扱い、現在と次の署名証明書とする。
証明書は読み取り可能な X.509 として比較し、単なる文字列の空白差を設定不一致にしない。
比較の計算は期待値と観測値を入力にして差分を返し、取得、認可、観測記録の保存はユースケースで編成する。
取り込む文書の形式、サイズの上限、欠落と不正値の拒否は TypeSpec と REQ-WSFEDERATION-001 に定める。

観測記録は WsFederation が所有し、対象テナントとドメイン、比較したローカル設定の版、鍵の識別子、取得時刻と確認時刻に結び付ける。
設定や鍵が変わった記録は現在の一致の証拠として表示しない。
手動で取り込んだ観測値であることを示し、継続監視や外部状態の保証として表示しない。
不足する権限、未実行、取得失敗、形式不正を「未適用」と同一視しない。

全自動方式は資格情報の保管と外部への書き込みを増やすため採らない。
管理者の「実行した」というチェックだけを適用確認にする方式も、値の不一致を検出できないため採らない。
既存の設定を移行するときは未確認として読み、再保存や外部変更を要求しない。
保存経路の所有者への集約は [[wi-85253-route-saml-and-wsfed-admin-saves-through-their-owners]] と整合させ、新たなモジュールやモジュール間依存は追加しない。

## 計画

1. REQ-WSFEDERATION-001 に、読み取り専用の再取得、観測値の比較、確認結果の鮮度、権限と拒否後の無変更を追加し、API の入出力を TypeSpec に定める。
2. 比較の型と操作、観測記録の保存先、設定変更時の失効条件を確定する。
3. API、PowerShell、管理画面、導入 runbook を順に実装する。

## タスク

- [ ] T001 [Spec] 反映確認と詳細の要件、入出力、上限、確認結果の有効範囲を定める。
- [ ] T002 [Acceptance] 保存だけでは適用済みにならず、再訪問でコマンドを読め、一致と不一致を観測できる試験を先に置く。
- [ ] T003 [App] 設定の再取得、比較、観測記録と PowerShell の生成を実装する。
- [ ] T004 [UI] 詳細、編集、反映確認、sync join と SSO の導入案内を実装する。
- [ ] T005 [Verify] 契約、権限、鮮度と画面遷移を検証し、手順を更新する。

## 検証

- `mise run test-go-package -- ./backend/wsfederation/...`：実装時に追加する要件を引き、欠落値、別テナント、証明書の差分、古い設定の観測値を誤って一致にしないことを確かめる。
- `mise run test-ui-e2e-file -- <追加する Entra 導入の spec>`：管理画面から保存し、再読み込み後の詳細とコマンド、未確認の表示、比較失敗時の入力保持まで確かめる。
- `mise run check-contract-drift` と `mise run check-published-api-compat`：管理 API の変更を確かめる。
- `mise run verify-changed`：実装の最終差分が届く層を検証する。実 Entra のコマンド実行は WI-79 で独立に確かめる。

## リスク

手動で持ち込む JSON は外部状態の真正性や、その後の変更まで保証しない。
確認の由来と時刻を表示し、不一致や読み取り失敗を成功にしない。
バックアップにはユーザーの情報や Graph の資格情報を含めず、ログにも残さない。
Graph の準備と復元手順は実装時点の Microsoft 公式資料に照合する。
