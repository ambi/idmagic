---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-22
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "現在状態の文書の配置と説明だけを整理し、利用者へ通知すべき製品機能、公開契約、移行手順は変えない。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - docs/README.md
    - docs/requirements/README.md
    - docs/requirements/functional.md
    - docs/requirements/quality.md
    - docs/design/README.md
    - docs/requirements/product-overview.md
    - docs/design/application/design-guidelines.md
    - docs/design/architecture/README.md
    - docs/design/architecture/system-boundary.md
    - docs/design/architecture/logical.md
    - docs/design/architecture/runtime.md
    - docs/design/architecture/deployment.md
    - docs/design/architecture/decisions.md
    - docs/design/verification/README.md
    - docs/design/verification/system-acceptance.md
    - docs/design/verification/security.md
    - docs/development/specification-first-workflow.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/development/continuous-integration.md
    - docs/domain/api-tokens/internals.md
    - docs/domain/application/internals.md
    - docs/domain/audit/internals.md
    - docs/domain/authentication/internals.md
    - docs/domain/authorization/internals.md
    - docs/domain/claim-mapping/internals.md
    - docs/domain/data-keys/internals.md
    - docs/domain/identity-governance/internals.md
    - docs/domain/identity-management/internals.md
    - docs/domain/jobs/internals.md
    - docs/domain/oauth2/internals.md
    - docs/domain/provisioning/internals.md
    - docs/domain/saml/internals.md
    - docs/domain/seeding/internals.md
    - docs/domain/sharedsignals/internals.md
    - docs/domain/signing-keys/internals.md
    - docs/domain/sourcing/internals.md
    - docs/domain/system/internals.md
    - docs/domain/tenancy/internals.md
    - docs/domain/workloadidentity/internals.md
    - docs/domain/ws-federation/internals.md
  typespec: []
  source:
    - tools/workspace/src/document-layout.ts
    - tools/check/src/check-documents.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/subdomain-classification.ts
    - tools/render-docs/src/main.ts
    - tools/render-docs/src/render.ts
  tests:
    - tools/check/src/document-layout-format.test.ts
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/docs-work-item-links.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
    - tools/check/src/terminology.test.ts
    - tools/render-docs/src/render.test.ts
    - tools/render-docs/src/documentation-quality.test.ts
  stop_before_reading:
    - backend
    - frontend
    - infra
    - spec
spec_impact:
  kind: none
  reason: "既存のプロダクト境界、実装、規範 ID、公開契約、数値目標を変えず、現在状態の文書の配置、索引、説明、用語を整理する。"
---

# システム文書の配置と説明を整理し、目的、要求、設計、検証の境界を明確にする

## 動機

`docs/README.md` は区分表の後に読み順と執筆規則を重ねており、入口として必要な情報より方法論の説明が長い。
区分表には要求文書、生成したリファレンス、文書フォーマットがなく、サイトの主要な入口を一覧できない。

`docs/design/README.md` は、要求、アーキテクチャ、設計、検証を設計文書の「層」として並べる一方、実体は `docs/requirements/`、`docs/architecture/`、`docs/design/`、`docs/verification/` に分かれている。
プロダクト概要も `docs/design/` にあり、何を求める文書と、どう実現する文書の境界がディレクトリ構成から読めない。
同じ README の「構成」と「領域」は重複しており、用語表には定義より運用ルールやツールの説明が混ざっている。

要求文書は、利用目的と担当 Context を示すだけで、パスワードなどの認証要素、上流から取り込む方式、下流へ同期する方式を一覧できない。
品質要求は一般的な名前である一方、数値を持つサービス目標はログインと OAuth 2.0、OpenID Connect に偏っている。
SAML IdP、管理機能、プロビジョニング、非同期処理などに数値目標がないことが、意図した範囲なのか未決定なのかも分からない。

アーキテクチャ文書と設計ガイドラインには、文書の目的を示さずに使う用語、複数の意味に読める説明、現在状態の理解に不要な方法論の説明が残る。
各 Context の `internals.md` は本文が日本語であるのに節見出しが英語であり、継続的インテグレーションの文書は「検証しないもの」だけを説明している。

## 対象範囲

- `docs/README.md` を区分表中心の入口にし、要求文書、設計文書、ドメイン設計文書、開発文書、運用文書、リファレンス、フォーマットを一つの表へ載せる。
- `docs/design/product-overview.md` を `docs/requirements/product-overview.md` へ移し、要求文書の入口から索引する。
- `docs/architecture/` を `docs/design/architecture/` へ、`docs/verification/` を `docs/design/verification/` へ移す。
- `docs/architecture/system-context.md` は `docs/design/architecture/system-boundary.md` へ改名し、文書名と見出しを「システム境界」にする。
- `docs/design/README.md` は、直下の設計領域だけを一つの表で索引する。
- `docs/requirements/constraints.md` を廃止し、なお有効な事実は既存の一次情報に存在することを確認する。
- `docs/requirements/functional.md` に、認証要素、上流からの取り込み、下流への同期を含む既存機能の分類を加える。
- `docs/requirements/quality.md` の測定用語、HTTP メソッド、サービス目標の対象範囲を明確にする。
- 移動後のアーキテクチャ文書を、各文書が何を説明するかを冒頭で判断できる形へ書き直す。
- `docs/design/application/design-guidelines.md` の Seam を平易な日本語で説明する。
- `docs/domain/*/internals.md` の人が読む見出しを日本語にする。
- `docs/development/continuous-integration.md` に、CI が検証する対象の説明を加える。
- 対象文書の長い段落から重複と方法論の説明を削り、列挙が主となる箇所は表または箇条書きにする。
- 文書配置を定める規約、生成サイト、リポジトリ検査、索引、現在状態の文書からのリンクを新しい配置へ合わせる。

## 対象外

- プロダクトの振る舞い、TypeSpec、規範シナリオ、標準仕様の採用内容、既存の数値目標の変更。
- SAML、管理 API、プロビジョニング、ジョブ、バッチなどに対する新しい SLO 値の決定と計装の追加。
  本項目では、数値目標がない領域を未決定として明示し、既存の目標が全機能を網羅しているとは説明しない。
- `internals.md` の設計内容の変更、英語の固有名詞、プロトコル名、コード識別子の翻訳。
- 対象として挙げた文書以外の内容を一律に書き直すこと。
- 完了済み work item の文章の推敲。

## 設計

### 文書の配置

要求は「何を必要とするか」、設計は「どう実現し、どう満たしたと判断するか」を扱う。
配置は次の形にする。

```text
docs/
  README.md
  requirements/
    README.md
    product-overview.md
    functional.md
    quality.md
  design/
    README.md
    architecture/
      README.md
      system-boundary.md
      logical.md
      runtime.md
      deployment.md
      decisions.md
    application/
    data/
    infrastructure/
    observability/
    performance/
    reliability/
    security/
    verification/
      README.md
      system-acceptance.md
      security.md
```

`constraints.md` は残さない。
現在の記載は、標準仕様、データの正本、同一オリジン、実行単位、シークレット、開発ツール、TypeSpec、データベーススキーマの各文書が扱う内容である。
有効な事実がほかに存在しない場合は、最も狭い既存の一次情報へ移すが、削除したページを別名で再作成しない。

`docs/design/README.md` には「構成」と「領域」を別々に置かない。
直下の `architecture/`、`application/`、`data/`、`infrastructure/`、`observability/`、`performance/`、`reliability/`、`security/`、`verification/` を一つの表で説明する。
要求とプロダクト概要は `docs/requirements/README.md` が索引する。

### 入口と要求文書

`docs/README.md` から「読み順」と「執筆上の境界」を削除する。
区分表に必要な入口を集約し、リファレンスには API、モデル、トレーサビリティから生成するページを、フォーマットには文書、仕様、work item の書式を載せる。

プロダクト概要の「課題」は、次の因果が短く読める説明へ書き直す。

1. 組織は人とワークロードのアイデンティティを複数のシステムで扱う。
2. 認証と同期が分断されると、ライフサイクル、権限変更、監査を一貫して扱えない。
3. IdMagic は認証、フェデレーション、プロビジョニング、委譲、監査をテナント境界の中で結び付ける。

同じ文書では、外部連携に対する曖昧な「契約」を「プロトコル」または「標準仕様」に置き換え、`runbook` を「運用手順書」と書く。
プロトコル上の契約や API 契約という明確な意味で使う「契約」は一括置換しない。

`functional.md` は、利用目的と担当 Context だけの表を、利用者が機能の有無を判断できる粒度へ広げる。
認証要素には、現在実装されているパスワード、TOTP、WebAuthn パスキー、リカバリーコードを載せる。
上流からの取り込みには、OIDC と SAML のフェデレーション、SCIM 2.0 のインバウンドプロビジョニングを載せる。
下流への同期には、SCIM 2.0 のアウトバウンドプロビジョニングを載せる。
詳細な振る舞いと API の置き場所を説明する末尾の二文は削除し、各行から担当する現在状態の文書へリンクする。

### 品質要求

`quality.md` の「スクレイプ対象」は、初出で Prometheus がメトリクスを定期収集する対象であると説明するか、「メトリクス収集対象」と書く。
エンドポイントを母集団にする行は HTTP メソッドを省略せず、トークンのレイテンシー、可用性、キャパシティをすべて `POST /token` と書く。

既存の数値目標は変更しない。
サービス目標の前に対象範囲の表を置き、次を区別する。

- 数値目標があるログイン、OAuth 2.0、OpenID Connect の経路。
- 外部 IdP から戻る OIDC と SAML のコールバックをまとめて測る目標。
- SAML IdP、管理 API、SCIM、監査、ワーカー、バッチなど、数値目標が未決定の領域。
- セキュリティとアクセシビリティのように、数値 SLO ではなく標準仕様と検証で扱う品質。

この表は未決定の数値を補わない。
現在の SLO が IdMagic 全体を代表するように読める見出しと説明を避け、対象外の領域を無言で除外しない。

### 設計用語

`docs/design/README.md` から、品質要求の一次情報を説明する一文、領域を跨ぐ問いの所有者を説明する二文、外来語を使う方針を説明する三文、用語検査の実装を説明する二文を削除する。

用語表は次の意味が読める定義へ直す。

| 用語 | 定義の要点 |
| --- | --- |
| デプロイメント | 実行単位を配置先へ割り当て、接続関係と環境差を定めた構成 |
| ランタイム | 稼働中の実行単位と、それらの間で行われる相互作用 |
| デプロイプロファイル | リポジトリが提供する配置構成の種類。実環境への適用実績と検証状況は別に示す |
| シークレット | 漏えいを防ぐためにアクセスを制限し、ソース管理へ保存しない機密値。機密性のない設定値は含めない |
| ガイドライン | 複数の設計で使う判断基準。個別の規則を定義または検査する場所は、その規則の担当文書が示す |

「ビューの名前」「通信へ写す」「選べる候補」「強制点」という表現は使わない。

`design-guidelines.md` では、最初に **差し替え点（Seam）** を、呼び出す側を変えずに実装または振る舞いを差し替えられる境界として説明する。
導入後は日本語の「差し替え点」を使い、Bounded Context の境界やポートとの違いを例で示す。

### アーキテクチャ文書

各文書の冒頭は、何を説明する文書なのかを一文で示す。
ISO/IEC/IEEE 42010 の用語を知らなければ本文を理解できない構成にはしない。

`system-boundary.md` は、IdMagic に含むもの、外部にあるもの、利用者と外部システムと運用者の責任分担を説明する。
DDD の Bounded Context と区別できない「システムコンテキスト」は、見出し、リンク文言、本文から除く。

`logical.md` から「System Context の画面」という文を削除する。
Context Map の前に、次の DDD 用語を簡潔に説明する表を置く。

| 用語 | この文書での意味 |
| --- | --- |
| Supplier | 下流へモデルまたは情報を提供する Context |
| Customer | Supplier が提供するものを利用する Context |
| Published Language | Context 間で共有する、公開された交換形式と語彙 |
| Open Host Service | 複数の利用側へ提供する公開された接続方法 |
| Anti-Corruption Layer | 外部の語彙を利用側のモデルへ変換する境界 |
| Subdomain | 業務能力を分けた問題領域 |
| Core | 製品の差別化に直接寄与する Subdomain |
| Supporting | Core を支える、製品固有の Subdomain |
| Generic | 製品固有でない一般的な能力の Subdomain |

Context の責務表の後にある、分類理由、公開言語、PostgreSQL の所有を一段落にまとめた説明は削除する。
必要な事実は、用語表、Context ごとの判断、データ設計がそれぞれ担当する。

`runtime.md` の「論理アーキテクチャを実行時のプロセスと通信へ写像する」は、稼働するプロセスと主要な相互作用を説明する文へ置き換える。
API、Worker、Batch を分ける説明は、入口とライフサイクルの違いだけを短く示す。
API を用途別に分割しない判断は担当する判断文書へ残し、ランタイム文書では繰り返さない。
`Frontend gateway` と `gateway` は、固有の別概念でないため「フロントエンドゲートウェイ」に揃える。

`deployment.md` の中継対象を説明する箇所は、「中継対象のパス一覧はこの文書へ複製せず、設定上の許可リストを参照する」と書く。
「散文」という語と、文書へ書かないことを強調する文は削除する。

### 日本語見出しと継続的インテグレーション

`docs/domain/*/internals.md` の英語見出しを日本語にする。
OAuth 2.0、OIDC、SAML、WebAuthn、PKCE、DPoP、CIBA、SCIM、CSV などの固有名詞と標準名は保つ。
見出しの変更でアンカーが変わるため、現在状態の文書からのリンクを同じ変更で更新する。

`continuous-integration.md` には「CI で検証するもの」を「CI で検証しないもの」より先に置く。
正確なジョブ一覧はワークフローを一次情報としたまま、次の分類を表で説明する。

- 仕様、文書、生成物、リンクの整合。
- Go、フロントエンド、リポジトリ内ツールの静的検査、テスト、ビルド。
- 公開契約、データベーススキーマ、依存関係、供給網の検査。
- 実配線を使うブラウザー E2E。

### 採らない方法

現在の四つの最上位区分を保ち、README の文章だけで要求、アーキテクチャ、設計、検証の関係を説明する案は採らない。
ディレクトリ構成と説明が異なる分類を示す状態が残るためである。

アーキテクチャと検証の文書を `docs/design/` 直下へ平置きする案は採らない。
同名になり得る `README.md` を持てず、文書群の境界と索引も失うため、`architecture/` と `verification/` の子ディレクトリにする。

品質要求の不足を埋めるために未実測の SLO 値を置く案は採らない。
整理と目標値の決定を同じ変更にすると、数値の根拠を文書表現の修正だけで済ませることになるためである。

## 計画

1. `DOCUMENTATION_GUIDE.md` と `SPECIFICATION_FORMAT.md` の配置、責務、リンクを新しい構成へ変更する。
2. 要求文書を移動して書き直し、`constraints.md` の各事実に既存の担当文書があることを確認してから削除する。
3. アーキテクチャと検証の文書を移動し、アーキテクチャ文書の導入、用語、不要な説明を整理する。
4. `docs/design/README.md` と設計ガイドラインを整理する。
5. `internals.md` の見出しと継続的インテグレーションの文書を直す。
6. 配置を列挙するリポジトリ検査、生成サイト、テスト、現在状態の文書のリンクを更新する。
7. 旧パス、削除対象の表現、英語の見出しが残っていないことを検索し、生成サイトの入口とリンクを検証する。

## タスク

- [x] T001 [Docs] 文書配置の正本を、要求と設計を分け、設計の配下へアーキテクチャと検証を置く構成へ更新する。
- [x] T002 [Docs] `docs/README.md` を区分表中心の入口へ整理し、要求、リファレンス、フォーマットを加える。
- [x] T003 [Docs] プロダクト概要を要求文書へ移し、課題、利用者、外部標準、運用手順書の説明を平易にする。
- [x] T004 [Docs] 機能要求へ認証要素、上流からの取り込み、下流への同期を加え、不要な末尾説明を削除する。
- [x] T005 [Docs] 品質要求の測定用語、HTTP メソッド、サービス目標の対象範囲と未決定領域を明確にする。
- [x] T006 [Docs] `constraints.md` の有効な事実の担当を確認し、重複を作らず同文書を削除する。
- [x] T007 [Docs] アーキテクチャと検証を設計の配下へ移し、システム境界、論理、ランタイム、デプロイメントの説明を整理する。
- [x] T008 [Docs] `docs/design/README.md` を直下の設計領域の索引へ改め、曖昧な用語と不要な文章を整理する。
- [x] T009 [Docs] 設計ガイドラインの Seam を「差し替え点」として説明する。
- [x] T010 [Docs] すべての `docs/domain/*/internals.md` の人が読む見出しを日本語にし、参照アンカーを更新する。
- [x] T011 [Docs] 継続的インテグレーションの文書へ、CI が検証する対象の分類を加える。
- [x] T012 [Tooling] 文書配置、生成サイト、リポジトリ検査、テスト、リンクを新しいパスへ合わせる。
- [x] T013 [Acceptance] 旧配置、削除対象の表現、説明のない用語、英語の見出しが残らず、生成サイトから各区分へ到達できることを確認する。
- [x] T014 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run check-spec`
- `mise run check-links`
- `mise run check-terminology`
- `mise run check-rendered-docs`
- `mise run test-tools`
- `mise run render-docs`
- `mise run verify`
- `git diff --check`

着手時の Acceptance RED は、旧パスと指摘された表現を完全一致で検索し、現在の文書と生成サイトに残っていることを記録する。
移動後は、旧パス、`constraints.md`、削除対象の文、説明のない `System Context` と `Frontend gateway`、`internals.md` の英語見出しが現在状態の文書に残っていないことを確認する。

実行可能なプロダクトの単体境界はない。
Unit RED の代替として、文書配置を列挙するリポジトリ内ツールのテストを新しい期待値へ先に変更し、旧配置に対して失敗することを確認する。

## リスク

文書の移動と見出しの翻訳は、多数の相対リンク、生成サイトの出力パス、リポジトリ検査の固定パスを壊し得る。
正本の配置を先に変更し、パスを認識するツールとリンクを同じ変更で追従させ、`check-links` と生成サイトのテストで検出する。

文章を短くする過程で、現在有効な設計判断まで削る恐れがある。
削除対象を、別の一次情報にある事実、方法論、重複、意味のない接続文に限り、ほかに所有者がない事実は担当文書へ移す。

機能要求へ具体例を足すと、未実装の機能を提供済みと読ませる恐れがある。
現在の実装と Context 文書で確認できる機能だけを載せ、候補や計画は記載しない。

品質要求の対象範囲を整理すると、未決定の SLO が増えたように見える恐れがある。
既存の数値を変えず、従来から数値目標がなかった領域を明示しただけであることを記す。

## 完了

- **Completed At**: 2026-09-22
- **Summary**:
  `mise run spec-diff -- main` は規範仕様に差分がないことを報告した。
  プロダクト概要を要求文書へ、アーキテクチャと検証を設計文書の配下へ移し、索引、説明、用語、リンク、検査ツールを新しい配置へ合わせた。
  機能要求には認証要素と同期方式を、品質要求には数値目標の対象範囲と未定義領域を明記した。
- **Acceptance RED Evidence**:
  - **Test**: 旧配置、削除対象の文、説明のない用語、英語見出しを検索する `rg` 検査
  - **Requirement**: N/A: 製品の振る舞いと規範仕様を変えない文書整理である。
  - **Observed Failure**: 実装前の検索は `docs/architecture/`、`docs/verification/`、`docs/design/product-overview.md`、`constraints.md`、指摘された表現、英語見出しを報告した。
  - **Detection Reason**: 旧配置または削除対象の説明を一つでも残す変更は、対応するパスか文字列へ一致する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/document-layout-format.test.ts`
  - **Requirement**: N/A: 実行可能なプロダクトの単体境界を持たない文書配置の変更である。
  - **Observed Failure**: 新しい要求・アーキテクチャ・検証のパスを期待するテストが、旧配置を返す `document-layout.ts` に対して失敗した。
  - **Detection Reason**: 文書配置の定義が旧パスへ戻ると、新しいパスを要求する表明が失敗する。
- **Change-Resistance Results**:
  Unit RED では、配置定義だけを旧状態に残す現実的な不整合を `document-layout-format.test.ts` が検出した。
  Acceptance 検査は、旧表現と英語だけの `internals.md` 見出しを変更前に検出し、変更後は一致なしになった。
- **Verification Results**:
  - `mise run check-links` - 成功（891 文書）
  - `mise run check-spec` - 成功（174 文書）
  - `mise run check-terminology` - 成功（234 文書）
  - `mise run test-tools` - 成功（588 テスト）
  - `mise run render-docs` - 成功（1,098 ページ）
  - `mise run verify` - 成功
  - `git diff --check` - 成功
