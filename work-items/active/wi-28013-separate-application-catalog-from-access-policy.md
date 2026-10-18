---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "Application の状態所有と管理の編成を分け、HTTP の応答、認可、割り当てとポリシーの判定、永続状態、イベントとトランザクションの順序を維持する。" }
---

# アプリケーションのカタログ管理を状態所有とアクセス判定から分ける

## 動機

Application は、プロトコルが利用する割り当てとポリシーの評価と、プロトコル設定を操作する管理を同居させていた。
SAML の辺をポートで切る当初案では、既存の検索順を保つためだけに更新用オブジェクトを追加し、相互依存の原因を残したまま型と委譲を増やす。
利用者の指示に従い、既存のモジュール分割を固定せず、管理の編成を分けて一方向の依存にする。

## 対象範囲

Application の HTTP の管理、ポータルとアイコンの配信を ApplicationCatalog へ移す。
Application は状態、保存、不変条件、更新操作、割り当てとポリシーの評価を所有し続ける。
SAML と WS-Fed の管理依存を Catalog に集約し、正式な HTTP の結線、責務表と生成する依存図を更新する。
過去の循環解消を確認し、再検討が必要なものを別項目にする。

## 対象外

HTTP と TypeSpec の契約、認証と認可、署名、プロファイル制約、イベント、原子性の変更。
OIDC の重複する管理型と変換層の除去は wi-86024、Tenancy の属性定義の複製の再設計は wi-94860 が扱う。
ほかの残存循環をこの作業へ混ぜない。

## 設計

[カタログ](../../docs/modules/application/features/catalog.md)、[割り当て](../../docs/modules/application/features/assignment.md)、[サインインポリシー](../../docs/modules/application/features/sign-in-policy.md)、[ポータル](../../docs/modules/application/features/portal.md)と、`spec/modules/application/main.tsp` を起点にする。
REQ-APPLICATION-001、005、006、007 を含む既存の HTTP テストは応答と保存状態を、ゲートのテストは拒否とステップアップを検査している。
比較基準は `fcbac4f6a` とする。

### 候補の比較

D1、D2、D3、D4、D7、D8 を適用する。

| 候補 | 隠す判断と変更の波及 | 採否 |
| --- | --- | --- |
| 現状を維持する | 状態所有と管理の編成が同じ単位なので、プロトコルの管理依存が評価側にも届く | 不採用 |
| 要求側の型と更新用オブジェクトを増やす | import は消えるが、項目の追加で HTTP、要求型、変換、所有型を同期する負担が増える | 不採用 |
| 管理の編成を ApplicationCatalog に分ける | 管理のワイヤ変換は Catalog、状態と評価は Application、プロトコルの保存と実行は各所有者に閉じる | 採用 |
| Application と全プロトコルを統合する | 管理以外にも異なる署名、通信と状態を同じ単位へ広げる | 不採用 |

プロトコルの設定項目を追加する変更は Catalog と当該プロトコルを、アクセス判定の変更は Application とその入力契約を見ればよい形にする。
新しいモジュールは SQL と独自の状態を持たず、既存の管理操作の編成を所有する。
モジュールの宣言だけを変更せず、本番の HTTP 実装と組み立ても移す。

### 公開操作と結線

Application の既存の `internal/usecases` を `application/usecases` に移し、同じ型、関数とエラーを状態所有者の公開操作として利用する。
ラッパー、コピーした入力型、管理 Repository は追加しない。
`CreateApplication(ctx, ApplicationDeps, CreateApplicationInput) (*domain.Application, error)`、`UpdateApplication`、`DeleteApplication`、割り当て、カテゴリ、ポリシーと並び順の操作の実装はそのまま保つ。
`Module.Gate` と `DesiredStateAssignments` も同じ操作群を使う。
HTTP が必要とする主体の存在確認は、既存の実装を返す `SubjectDirectory(users, groups) ports.SubjectDirectory` を組み立て地点から共有する。

ApplicationCatalog の `Register(g *echo.Group, deps Dependencies)` は既存の HTTP の依存を受け取り、同じ経路を登録する。
HTTP の本体とテストを `backend/applicationcatalog/internal/handlers_http` へ移す。
Application のメモリ保存先を直接組み立てていた六つの fixture は同じ公開コンストラクターの保存先を利用し、未配線だった個別ポリシーは未配線のまま保つ。
テストの入力と表明、特性化テストの本文は変えない。
OIDC 管理の要求型は Catalog の `ports` へ移し、小さな `ClientNames` の照会だけを Application に残す。
既存の OIDC 変換層は新しい型の参照へ接続し直し、除去は独立した wi-86024 で行う。

SAML と WS-Fed の処理は今回移動だけを行う。
証明書と IdP プロファイルの拒否、JSON 解析前の検索、更新時刻、空の配列、途中失敗の状態を含む処理の順序を変えない。
SP と RP の Repository 実装、Application の PostgreSQL の protocol linker と同一 Tx の保証を変更しない。
Catalog は Application の公開更新操作を利用し、SQL の書き込み先とテーブル所有者は増やさない。
残るプロトコルの検証処理の配置は、Catalog の分離後に各公開操作を改善するときに判断する。

### 過去の変更の確認

wi-60634 は管理型と同じ内容の変換が増えたため wi-86024、wi-53317 は属性定義の二重表現と列挙の文字列化が増えたため wi-94860 にする。
wi-51457 は認証方式の解釈を Authentication に戻す小さな読み取り契約なので維持する。
wi-61185 のパスワード既定値、wi-64444 の SCIM の取り込み元の照会、wi-49676 の監査の名前の照会、wi-73604 と wi-73872 のテナント ID と解決済み設定の分離は、必要な知識の縮小または所有者への集約があり、同じ形の大きな管理契約を複製していないため今回の再実装対象にしない。

### 安全性と互換性

認可、CSRF、テナント確認とフェイルクローズの分岐を移動先でも同じ操作列で実行する。
API、DB と保存データの移行は不要で、ロールバックはコードの変更を戻せばよい。
循環を再計測し、実際に消えた違反だけを台帳から削除する。
WS-Fed の独立したポート追加を計画した wi-61016 は、この分離で目的を達成できたことを確認して中止する。

## 計画

設計と再検討項目を確定し、既存の HTTP の96テストの基準結果を得る。
状態所有者の公開操作と Catalog の分離、既存テストの移動、正式な結線を一つの構造変更として行う。
責務表と所有文書を同期し、実測した負債だけを除去する。
全体の検証と差分レビューの後に正式コミットと完了記録を作る。

## タスク

- [x] T001 [Design] 責務、公開操作、作用の比較を記し、過去分の再検討を起票する。
- [x] T002 [App] Catalog を分離し、既存の状態更新とアクセス判定を保って結線する。
- [x] T003 [Docs] 現在の設計と依存図を同期し、負債と重複した計画を整理する。
- [x] T004 [Verify] HTTP、拒否後の状態、ゲート、正式な結線と仕様差分を検査する。

## 検証

既存テストを移動するため、テストのパスが変わらない構造変更の免除には該当しない。
実装前の HTTP テストの成功を基準に、登録の欠落が同じテストで失敗することを確認して戻す。
実装後に Catalog の HTTP、Application の公開操作とゲート、PostgreSQL の protocol linker を既存テストで検査する。
HTTP の結線、割り当ての依存とポリシーの拒否を独立した故障として選び、変更行に計算の変更がなければ純粋ロジック全体を再変異せず、これらの故障の検出を確かめる。
新しい解析、正規化と値の相互変換は追加しないため、ファズ対象を増やさない。

- `mise run test-go-package -- ./backend/applicationcatalog/internal/handlers_http`：既存の応答、保存値、無作用とクレームの下限を検査する。
- `mise run test-go-changed -- fcbac4f6a`：正式な組み立て、プロトコルと逆依存を race 付きで検査する。HTTP の待受けを使うためサンドボックス外で実行する。
- `mise run lint-go`、`mise run check`、`mise run check-published-api-compat`、`mise run render-docs`：配置、型、公開契約と所有文書を検査する。
- `mise run check-boundary-debt-ratchet -- fcbac4f6a` と `mise run spec-diff fcbac4f6a`：新しい違反がなく規範を変えていないことを確認する。

## リスク

HTTP の移動により登録が欠落すること、依存の取り違えで割り当てとポリシーを迂回することが主な故障である。
既存の正式な HTTP スタックと拒否のテストを保ち、コードの移動と処理の変更を混ぜない。
公開操作のパッケージを追加するため、内部の SQL と保存実装を公開範囲へ含めない。

## 要約

Application の状態、更新とアクセス判定から、HTTP とプロトコル設定の管理を ApplicationCatalog へ分離した。
Application の8ファイルのユースケースは内容を変えずに公開パッケージへ移し、Catalog が同じ型と操作を直接利用する。
新しい管理用の入力型、Repository、SAML の更新用オブジェクトと変換層は追加していない。
SQL、テーブル所有者、protocol linker とトランザクションの境界も変えていない。
既存テストの入力、表明と特性化テストの本文を保ち、パッケージの参照と六つのメモリ fixture の組み立てだけを移動先に合わせた。

循環の件数と設計改善を区別し、責務の分離で SAML と WS-Fed を含む12件が循環から外れたことを再計測した。
台帳は29件から17件となり、重複する WS-Fed のポート追加案 wi-61016 を中止した。
wi-60634 の OIDC 管理の重複する型と委譲は wi-86024、wi-53317 の属性定義と相互変換は wi-94860 に再検討を起票した。
それ以外の確認した循環解消は、知識の縮小または所有者への集約があるため今回の再実装対象にしていない。

移動前後の Catalog の既存96テストと、変更パッケージと逆依存の race 付き2,218テストが成功し、後者の失敗、スキップと未実行は0件だった。
Application 全体の追加検査は217テストが成功し、既存の `FuzzClientIPAllowed` の不正 CIDR の二つのシードとその親の3件だけがスキップされた。
PostgreSQL の Application と protocol linker の検査も成功した。
SAML の経路の削除、Catalog の入口の登録の省略、主体照会の未配線、個別ポリシーの未配線、デフォルトポリシーの未配線を独立に注入し、変更していない既存テストがそれぞれ失敗することを確認して復旧した。
純粋な計算の実装は変更していないため、変更耐性は新しい結線の故障で確認し、既存ロジック全体の再変異とファズ対象の追加は行っていない。

七つの視点で、管理と評価の担当、同じ入力型の利用、公開操作と保存実装の分離、fixture の等価性と結線をレビューした。
`lint-go`、`check fcbac4f6a`、文書生成と公開 API の互換性ゲートは成功し、`spec-diff fcbac4f6a` は規範の変更なしだった。
公開前なので互換性ゲートは公開済み契約との比較を行っていない。
HTTP の静的解析に既存の未解析操作が残るため、全操作を静的に検証済みとは扱わない。
OIDC 管理と属性定義の重複の除去は、起票した二つの未着手項目に残す。
