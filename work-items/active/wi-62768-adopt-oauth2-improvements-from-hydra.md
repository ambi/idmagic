---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-035 }
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-008 }
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-032 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-006 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-018 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-011 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-019 }
  - { path: docs/modules/oauth2/features/logout.md, requirement: REQ-OAUTH2-025 }
---

# Ory Hydra を参照して OAuth2 の同意とトークンのライフサイクルを改善する

## 動機

[Ory Hydra](https://github.com/ory/hydra) の機能と設計を比較し、IdMagic の OAuth2 と OIDC に取り入れる改善を実装する。
IdMagic は認証、同意、トークン発行、失効をすでに持つため、比較では機能名の有無に加えて、利用者が制御できる範囲、状態の関連付け、失敗時の保証、運用の継続性を見る。

起票では粗い比較と候補の抽出までを行う。
実装への着手時に Hydra のソース、標準仕様、IdMagic の仕様と実装を詳しく調べ直し、採用する改善を確定してから仕様、実装、検証を進める。
候補表は未実装の機能一覧や確定仕様を意味しない。

## 対象範囲

- クライアントとグラントに応じたトークンの有効期間、同意の記憶と再利用、同意とトークンとセッションの失効の連携を比較し、採用する改善を実装する。
- ログインと同意の継続処理における要求の束縛、単回利用、期限と、短命なプロトコル状態の後始末を比較し、既存機構の不足を改善する。
- 採用した改善を、管理 API、必要な管理画面、永続化、監査、運用経路までつなぐ。
- Hydra のテストから参考になる成功例と拒否例を抽出し、IdMagic の要件を根拠に回帰検証を加える。

## 対象外

- Hydra の導入、IdMagic の OAuth2 実装の置き換え、Hydra 互換の管理 API、別サービスとしてのログイン基盤の新設。
- Ory Network や Enterprise 固有のサービスを、OSS Hydra の機能として取り込むこと。
- Implicit Flow、Hybrid Flow、Password Grant など、IdMagic が採用していない方式を比較対象の存在だけを理由に追加すること。
- 任意の外部 HTTP フックやスクリプトによるクレームの書き換え。具体的な利用要件が出た場合は、[ClaimMapping の宣言的な規則](../../docs/modules/claim-mapping/README.md#設計の要点)との比較を別の作業項目で行う。
- リフレッシュの再利用を許す猶予期間の導入。IdMagic は[並行利用も再利用として扱う](../../docs/modules/oauth2/features/token.md#リフレッシュトークンのローテーションと再利用検知)方針を持ち、この比較ではその保証を維持する。
- pairwise subject、PKCE 方針、既知の同意状態遷移の修正、リフレッシュ要求の範囲縮小、外部準拠検証の CI 基盤を重複して実装すること。担当項目は設計の対応表に示す。

## 設計

### 比較の粒度

IdMagic の[プロダクトの目的](../../docs/requirements/product-overview.md)と[OAuth2 の責務](../../docs/modules/oauth2/README.md#責務と境界)を採用判断の入力とする。
Hydra と同じ設定や構成を増やすこと自体は目的にしない。
起票時に確認した IdMagic の状態は主に仕様上の宣言であり、実装とテストの充足は着手時に確認する。

Hydra の調査日は 2026-11-06 とし、公開ソースの参照を `master` の commit `4174065ffb052799890f7480f5360a877a67ffc1` に固定した。
[Hydra の README](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/README.md)は OAuth2 と OIDC のサーバーとしての役割と、外部のログインと同意のアプリケーションとの接続を説明する。
ユーザー管理や MFA も持つ IdMagic との比較では、この責務の違いを前提にする。
以下は起票時の確認であり、安定版の全機能や実運用での保証を検証したものではない。

### 起票時の採用候補

| 候補 | Hydra の参考要素と出典 | IdMagic の現状と取り入れる狙い | 実装時に確かめること |
| --- | --- | --- | --- |
| クライアントとグラントに応じた有効期間 | [クライアントにグラント別のトークン寿命を持たせる](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/client/client.go) | [有効期間の設計](../../docs/modules/oauth2/features/token.md#有効期間セキュリティ保持期間の設定)と [JWT 署名処理](../../backend/oauth2/internal/tokens_jose/jwt_signer.go)には共通の有効期間がある。連携先に応じた短縮と、運用者が上限を制御できる設定を候補にする | テナント、クライアント、Agent の既存上限の優先順位、未指定値、許容範囲、発行グラントごとの適用、`exp` と `expires_in` の一致。更新後の既発行トークンへの影響と移行 |
| 同意の記憶と再利用の制御 | [同意の `remember`、`remember_for`、付与するスコープと audience を別々に表す](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/flow/consent_types.go) | [REQ-OAUTH2-008](../../docs/modules/oauth2/features/consent.md#req-oauth2-008-既存同意の有無に応じて同意画面を出し分ける)は同意を 365 日で保存する。同意の記憶の有無と期間、実際に許可した範囲の扱いを候補にする | 一回限りの同意、既存同意への追加、要求範囲と許可範囲、対象リソース、`prompt`、第一者クライアント、同意画面の操作と監査。既存の権限縮小規則との整合 |
| 同意の撤回と派生トークンの失効の連携 | [同意要求 ID などの単位で失効し、関連トークンの失効範囲も指定する](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/consent/handler.go) | [同意の撤回](../../docs/modules/oauth2/features/consent.md#本人による同意の撤回)、[トークンの失効](../../docs/modules/oauth2/features/token.md#クライアントによるトークンの失効)、[ログアウト](../../docs/modules/oauth2/features/logout.md)は個別に定義されている。関連する状態を追跡し、撤回後の利用を止める範囲を明確にする | 同意、認可、トークンファミリー、`sid` の関連、撤回後のリフレッシュとイントロスペクション、JWT をオフライン検証する RP の限界、同時発行との競合。無関係な端末、クライアント、テナントへの影響 |
| ログインと同意の継続処理の保証 | [challenge への承認と拒否、verifier による再開、使用済み同意への拒否を分ける](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/consent/handler.go) | [認可の状態遷移](../../docs/modules/oauth2/features/authorization.md#authorizationcodeflow)と [Cookie による継続処理](../../backend/oauth2/internal/handlers_http/authorize_transaction.go)がある。Hydra の要求と再開の分離を参考に、既存経路の不足を調べる | 要求と利用者とクライアントの束縛、CSRF、期限切れ、許可と拒否の再送、別の要求への取り違え、並行タブ、途中失敗後の再開。新しい公開 API が必要か、既存の状態管理で満たせるか |
| 短命な認可状態の後始末 | [非アクティブなトークンを基準時刻、件数上限、バッチの大きさを指定して削除する](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/persistence/sql/persister_oauth2.go) | [データのライフサイクル](../../docs/design/data/lifecycle.md#データの区分と消え方)は期限切れによる無効化と掃除を区別する。Hydra の後始末を参考に、既存の削除経路で不足する対象を補う | 認可コード、PAR、認可要求、デバイスコード、使用済みリフレッシュトークンの保持理由と削除順序。再利用検知に必要な記録、監査と同意の保持、バッチ件数、再実行、処理中の要求との競合 |

検証の参考には、Hydra の[外部適合性スイートの実行コード](https://github.com/ory/hydra/blob/4174065ffb052799890f7480f5360a877a67ffc1/test/conformance/run_test.go)も含める。
IdMagic の期待結果は Hydra の応答をコピーして作らず、採用する標準要件と製品要件から定める。

### 既存の作業項目との分担

| 関連項目 | この項目での扱い |
| --- | --- |
| [[wi-61672-decide-whether-pkce-is-required-per-client]] | PKCE の必須化方針は担当項目で決める。Hydra の設定を理由に方針を重ねて変更しない |
| [[wi-54733-move-authorization-requests-through-the-consent-states]] | 既知の同意状態遷移の不一致は担当項目で直す。継続処理を実装する前に最新状態を確認する |
| [[wi-11707-honor-the-scope-and-resource-of-a-refresh-request]] | リフレッシュ要求での範囲縮小は担当項目で実装する。同意と有効期間の改善は確定した契約と整合させる |
| [[wi-298-pairwise-subject-identifiers]] | Hydra の subject の設計は担当項目への調査材料とし、この項目では実装しない |
| [[wi-33-protocol-conformance-ci]] | 外部準拠検証の基盤は担当項目で作る。この項目では採用する振る舞いの回帰テストを担当する |

起票時点では、どの候補も上記の項目全体の完了を必須とはしていない。
採用内容が既存項目の変更に依存すると分かったら、実装前に分担と `depends_on` を更新する。

### 採用する設計の決め方

既存機構の拡張を先に比較し、新しい状態、ポート、設定は利用者に必要な振る舞いを実現する場合に限って追加する。
OAuth2 のプロトコル状態、Authentication のセッション、ClaimMapping の開示規則、SigningKeys の署名鍵という担当を起点にする。
現時点ではモジュール境界の変更を決めていない。
着手時に依存、公開範囲、保存の所有者、組み立て地点の変更が必要と分かったら、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)による候補比較を実装前にこの節へ記す。

実装前に、候補ごとに採用、不採用、既存機能で充足、別項目で対応の結論と理由を残す。
採用分について要件の差分、公開契約、主要な型と操作、時刻と永続化の境界、競合と失敗時の結果、互換性、移行とロールバックを具体化する。
候補表の問いは着手時の調査課題であり、実装に進む条件として解決する。
後回しにする候補は対象外へ移す。

## 計画

1. Hydra の参照リビジョンと OSS に含まれる範囲を確認し直し、候補に対応するソース、テスト、標準仕様を読む。IdMagic の実装と既存項目の進捗も照合する。
2. 候補の採否と実装範囲を確定し、`affected_spec` と依存項目を更新する。採用する振る舞いの変更を所有する仕様、TypeSpec、設計へ反映する。
3. 観測結果と故障を `primary_use_cases` に具体化する。選んだ境界で RED を確認し、一つの振る舞いずつ実装して GREEN にする。
4. 必要な永続化、管理操作、画面、監査、運用経路を接続し、互換性、移行と失敗時の振る舞いを検証する。
5. 採用した改善の実装と検証を完了し、現在の仕様と設計へ結論を残す。調査結果の列挙だけで実装を完了扱いにしない。

## タスク

- [ ] T001 [Research] Hydra と IdMagic のソースとテストを詳しく比較し、候補ごとの採否と理由を設計へ記す。
- [ ] T002 [Spec] 採用分の要件差分、TypeSpec、設計、移行方針を確定し、`affected_spec` と `primary_use_cases` を更新する。
- [ ] T003 [Acceptance] 採用する各ユースケースの成功、拒否、期限と撤回後の観測結果を検証するテストで RED を確認する。
- [ ] T004 [App] 採用した有効期間、同意、失効と継続処理の改善を、既存の所有モジュールで実装する。T002 で決めた要件と対応テストを各実装タスクへ追記する。
- [ ] T005 [Adapters] 必要な管理 API、画面、保存とバッチの配線を実装し、テナント境界、並行更新、部分失敗の検証を接続する。
- [ ] T006 [Verify] 変更に応じた検査を実行し、採用した改善の実装、移行と運用の成立を確認する。

## 検証

起票時は `mise run check-work-items` で形式と規範参照を、`mise run check-links` で内部リンクを検査する。
実装時の検証対象と追加タスクは T002 で確定する。

- `mise run check-spec`：採用する要件と TypeSpec を変更したとき、その形式と整合を確かめる。
- `mise run test-go-package -- <変更パッケージ>`：期限の境界、単回利用、権限の上限、撤回後の拒否、拒否後の無作用を、故障を観測できる最小の境界で確かめる。
- `mise run test-go-package -- <変更パッケージ> --race`：同じ要求の消費、失効と発行、削除と利用の並行処理を変更した場合に実行する。
- `mise run test-ui-e2e-file -- <対象ファイル>`：同意や管理画面を変える場合、正式な準備から保存、発行や拒否までの接続を確認する。
- `mise run verify-changed -- <着手時の基点>`：コード変更を終えた時点で、実際の差分に応じた検査を選ぶ。
- `mise run spec-diff`：完了時の要約を、実際の仕様差分から作る。

HTTP、DB、ブラウザーを使う検証は、リポジトリの実行規則に従いサンドボックス外で実行する。
DB を使う保存契約では、再利用検知に必要な記録が保持され、後始末後も無効な要求を受理しないことを確かめる。

## リスク

- 有効期間や同意の再利用を変えると、既存の連携が想定する再認証の頻度や継続利用に影響する。未指定時の互換性と、安全側の上限を先に決める。
- 撤回、ログアウト、失効の関連を広げると、無関係なセッションまで止める恐れがある。関連付けるキーと失効の範囲を仕様で定め、別の端末、クライアント、テナントが影響を受けないことも検証する。
- 期限切れデータの削除が早すぎると、再利用検知や監査の証拠を失う。無効化と物理削除の時点を分け、保持理由ごとに削除の条件を決める。
- Hydra の既定値や挙動は IdMagic の要件を保証しない。比較結果は採用の材料とし、標準仕様と IdMagic の利用要件から期待結果を独立に定める。
