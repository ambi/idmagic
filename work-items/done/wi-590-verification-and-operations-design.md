---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p2
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
spec_impact:
  kind: none
  reason: "検証設計と運用文書の担当範囲と内容を書き下すだけで、テスト、CI、運用体制、SLO の値は変えない。"
documentation_impact:
  level: none
  reason: "増えるのは開発者と運用者が読む検証設計と運用文書だけで、利用者が観測する振る舞い、API、設定キー、SLO の値はどれも変わらない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - docs/verification/README.md
    - docs/verification/system-acceptance.md
    - docs/operations/README.md
    - docs/operations/service-management.md
    - docs/operations/maintenance.md
    - docs/development/testing.md
    - docs/development/release.md
    - docs/requirements/quality.md
    - docs/design/security/threat-model.md
    - docs/design/reliability/README.md
    - docs/design/observability/monitoring.md
    - docs/README.md
    - DOCUMENTATION_GUIDE.md
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
  tests: []
  stop_before_reading:
    - backend
    - frontend
    - docs/contexts
    - spec
---

# 検証設計の担当範囲を決め、運用文書に不足する設計を書く

## Motivation

[検証設計](../../docs/verification/README.md)は 15 行の索引と、その子である [システム受入れ設計](../../docs/verification/system-acceptance.md) 17 行だけからなる。
索引の表は「対象」と「主な証拠」を六行で対応させるが、どの要求にどの証拠が要るかという対応は持たない。
子がシステム受入れ一つだけなので、このディレクトリがシステム受入れのためだけに存在するように見える。

一方で、テストの水準（単体、アダプター統合、受け入れ、E2E、運用検証）、テストダブルの選び方、プロパティテストと変異テストの適用条件は、既に [テスト方針](../../docs/development/testing.md) が 60 行以上をかけて持っている。
つまり「ユニットテスト設計」「E2E テスト設計」に相当するものは存在するが、検証設計の側からそこへ到達する経路が無い。
検証設計を開いた人は、テストの設計がこのプロダクトに無いように見える。

運用文書も薄い。
[サービス管理](../../docs/operations/service-management.md)は 19 行で、SLO の評価、インシデント時の役割、変更と継続性を各三文ずつ述べる。
当番の体制、エスカレーションの経路、宣言の水準、ポストモーテムの扱い、変更の分類と承認は書かれていない。
[保守](../../docs/operations/maintenance.md)は 17 行で、定期作業を一文で列挙し、実行頻度と担当は「採用する運用環境で定める」として空欄のままである。

## Scope

- [検証設計](../../docs/verification/README.md)の担当範囲を冒頭で宣言し、テストの水準設計が [テスト方針](../../docs/development/testing.md)にあることへの到達経路を置く。
- 要求と証拠の対応を、要求 ID、証拠の種類、実行するタスク、合否条件、現在の状態の表として書く。
- システム境界でしか確かめられない検証のうち、現在どの文書も持っていないもの（セキュリティ検証、可用性と復旧の試験、宣言的なファイルの検証）の設計を足す。
- 受入れと、許可しないリクエストの確認対象を、使う文書の冒頭で書き下す。
- [サービス管理](../../docs/operations/service-management.md)に、当番の体制、エスカレーション、宣言の水準、変更の分類、ポストモーテムの扱いを書く。
- [保守](../../docs/operations/maintenance.md)に、定期作業ごとの契機、頻度の決め方、担当の分界、廃止と引渡しの設計を書く。

## Out of Scope

- テストの追加と CI の変更。
- テスト水準の設計そのもの。[テスト方針](../../docs/development/testing.md)が正本であり、ここへ写さない。
- 実作業の手順。[runbook](../../docs/runbooks/) が持つ。runbook の整備は [[wi-290-alert-runbook-catalog-and-on-call-operations]] が扱う。
- SLO、RPO、RTO の値。[品質要求](../../docs/requirements/quality.md)が正本である。
- 実際の当番体制と担当者の割り当て。運用体制が決まってから決める。
- 検証をいつ誰が実施したかの記録。設計文書は実行環境だけを区別し、実施は[保守](../../docs/operations/maintenance.md)の定期作業が持つ。
- `mise.toml` への PostgreSQL クライアントの固定。復元試験の実行条件は記述するにとどめる。
- 負荷試験と高可用性の実施。[[wi-282-staging-load-testing-and-capacity-validation]] と [[wi-165-high-availability-and-failover-resilience-topology]] が扱う。

## Design

### 「ユニットテスト設計」を検証設計へ新設しない

検証設計と開発文書の分かれ目は `DOCUMENTATION_GUIDE.md` §4.13 と §8.4 が定めている。
検証設計は「どの証拠で要求を満たしたと判断するか」を持ち、開発文書は「どう書き、どう走らせるか」を持つ。
テストの水準、境界、実物にする依存、テストダブルの選び方は後者で、既に [テスト方針](../../docs/development/testing.md)にある。
検証設計に「ユニットテスト設計」を新設すると、同じ観点の正本が二つになり、片方だけが更新される。

現在の問題は、テスト設計が無いことではなく、次の二つである。

- 検証設計が自分の担当範囲を宣言していないので、何がここに無いのかが読み手に伝わらない。
- 要求から証拠への対応が、どこにも表として存在しない。

したがって足すのは、担当範囲の宣言と、要求と証拠の対応表である。

| 列 | 内容 |
| --- | --- |
| 要求 | `REQ-*`、`SLO-*`、`CAP-*`、標準の規範 ID のいずれか |
| 証拠 | 静的解析、単体、アダプター統合、受け入れ、E2E、負荷、障害試験、復元試験、運用確認のどれか |
| 実行 | `mise` タスクまたは CI の名前 |
| 合否 | 何が起きたら満たしたと判断するか |
| 現状 | 実行できる、環境が無い、未着手のどれか |

全要求を一行ずつ並べるのではない。要求の群ごとに一行とし、個々の `REQ-*` とテストの対応はテスト側の `//spec:covers` が持つ。
この表が答えるのは「この種類の要求は何で確かめるか」であって、「この要求のテストはどれか」ではない。後者を表にすると、テストが増減するたびに古くなる。

### システム境界でしか確かめられない検証

`verification/` の子として現在あるのはシステム受入れだけである。
`DOCUMENTATION_GUIDE.md` §4.13 は「機能の正常経路だけでなく、品質要求、構成要素間の接続、縮退、切替、復元をシステム境界で確認する」と定めており、次が現在どの文書にも無い。

- **セキュリティ検証**。[脅威モデル](../../docs/design/security/threat-model.md)の脅威 ID と、それに応える制御の検証手段の対応。拒否が防いだ作用を確かめる試験、依存監査、供給鎖の検証。
- **可用性と復旧の試験**。何を起こし、何を観測し、どこまで戻ったら成功とするか。頻度。
- **運用検証**。構成ファイルのレンダリング、監視ルールの妥当性、スキーマ収束、設定リファレンスの乖離。

配置は次のとおりとする。

| 検証 | 置き場所 | 理由 |
| --- | --- | --- |
| セキュリティ検証 | 新設する [セキュリティ検証設計](../../docs/verification/security.md) | 脅威 ID と証拠の対応、許可しない操作の確認、依存監査、供給鎖の四つを持ち、受入れの節に収まらない |
| 可用性と復旧の試験 | [システム受入れ設計](../../docs/verification/system-acceptance.md)の節 | 品質受入れが既に可用性と復旧を対象にしており、何を起こし、何を見て、何をもって合格とするかはその具体化である |
| 運用検証 | [検証設計](../../docs/verification/README.md)の節 | 独立した文書にはしない。理由は次の節に書く |

### 運用検証を独立した文書にしない

宣言的なファイルの検査は、独立した文書を立てるだけの担当範囲を持たない。

置き場所の候補を順に見ると、そのことが分かる。
`check-k8s`、`check-compose`、`check-monitoring`、`check-schema`、`check-config-reference`、`check-route-reference` は、いずれもデプロイ前にリポジトリ上で走らせる検査であり、稼働中のシステムに対する作業は一つも無い。
したがって「稼働後のサービス管理と保守」を持つ `docs/operations/` には収まらない。
いつ走らせるかは [継続的インテグレーション](../../docs/development/continuous-integration.md)が既に正本として持ち、「運用検証」という水準の定義は [テスト方針](../../docs/development/testing.md)の水準表が持つ。

残るのは対象ごとの合否条件だけで、これは要求と証拠の対応そのものである。
よって検証設計の節として置き、実行のタイミングは継続的インテグレーションを参照する。

### 進捗ではなく実行環境を列にする

当初は「実行できる、環境が無い、未着手」の三値で現状を書くつもりだった。
これは進捗であり、いつ誰が実施したかの記録である。
設計文書が答えるべきは、その証拠を得るために何を用意するかであって、いま緑かどうかではない。

そこで列は実行環境とし、次の三値を採る。

| 値 | 意味 |
| --- | --- |
| リポジトリ内 | 手元と CI で、このリポジトリにあるものだけを使って実行できる |
| 実環境が必要 | 本番または本番相当の環境がなければ実行できない |
| 手段が未定 | 証拠の作り方をまだ決めていない |

`DOCUMENTATION_GUIDE.md` §4.13 が求める「実環境が必要で未実施の検証を設計上の仮定と区別して残す」ことは、この列が引き受ける。

値は実際に走らせて決める。
確かめた結果は次のとおりである。

| タスク | 結果 |
| --- | --- |
| `check-security-controls`、`check-slo-references`、`check-config-reference`、`check-route-reference`、`lint-go` | 成功 |
| `check-k8s`、`check-monitoring`、`check-k6`、`check-compose`、`check-schema` | 成功。いずれも Docker を要する |
| `restore-drill` | 失敗。Docker に加えて PostgreSQL クライアントを要するが、`mise.toml` は `psql` を固定しておらず、シムが版を解決できない |

`restore-drill` の失敗は検証設計の欠落ではなく、復元試験の実行条件が固定されていないことによる。
検証設計には、ローカルの復元試験が Docker と PostgreSQL クライアントを要することを実行環境として書く。
`mise.toml` へ固定を足すのはこの work item の対象外とし、別の記録が扱う。

### 語の選び方

「演習」と「ドリル」は使わず、「障害試験」と「復元試験」で書く。
[リカバリ設計](../../docs/design/reliability/recovery.md)の「復元試験」と[可用性設計](../../docs/design/reliability/availability.md)の「障害試験」が既にあり、同じものを外来語で呼び直す理由が無い。
[[wi-583-normalize-design-document-terminology]] が「訓練」の行き先として選んだ「ドリル」は、日本語として不自然なので採用語を差し替える。

「拒否」は名詞として立てず、「許可しないリクエスト」「操作を実行せずにエラーを返す」のように書く。
名詞にすると「拒否の受入れ」「拒否が防いだ作用」のような、何を指すのか読めない言い回しになる。

受入れと、許可しないリクエストで何を確かめるかは、最初に使う文書の冒頭で書き下す。
書かないと、確認がレスポンスの確認に縮む。

### 頻度と担当は運用文書が持つ

検証設計は何を確かめ、何をもって合格とするかを持ち、いつ誰が実施するかは持たない。
試験の頻度を検証設計へ書くと、運用体制が決まるたびに設計文書を書き換えることになる。
[保守](../../docs/operations/maintenance.md)の定期作業は、頻度を決める条件と担当の分界を列で持っているので、試験はその行として並ぶ。

### サービス管理と保守

サービス管理へ足すもの。

- **当番とエスカレーション**。役ごとの責任（指揮、実作業、記録、連絡）は既にある。足すのは、誰がページを受け、何分で応答し、応答が無いときにどこへ上がるか。役は人ではなく役で書く。
- **宣言の水準**。どの影響範囲でインシデントを宣言するか。水準ごとに誰が集まるか。
- **変更の分類**。通常、緊急、標準の三つに分け、それぞれの承認と観測の要求を書く。段階的なデプロイと後退の基準は [リリース手順](../../docs/development/release.md)が持つので参照する。
- **ポストモーテム**。いつ書くか、何を書くか、恒久対策が規則になった時点で仕様へ反映すること。個人の責任を問う書き方をしないこと。
- **error budget を使い切ったときの扱い**。現在「未確定であり [[wi-419-quantification-beyond-performance]] が扱う」と書かれている。扱いが決まるまでは、決まっていないことによって何が起きるかを書く。

保守へ足すもの。

- **定期作業の表**。作業、契機、頻度の決め方、担当の分界（プロダクト側か運用基盤側か）、対応する runbook。頻度の値は運用環境で決まるが、**何を契機に決めるか**はここで決められる。証明書の期限なら発行間隔、依存更新なら脆弱性の通知と定期更新の両方、といった形で書く。
- **依存更新**。更新の範囲、検証、後退の扱い。
- **廃止と引渡し**。機能、データ、鍵、外部連携を廃止するときの、保持、失効、エクスポート、監査、シークレットの廃棄、運用責任の終了を一つの計画として扱う既存の記述を、段階の表へ展開する。

## Plan

1. 検証設計の担当範囲を決め、テスト方針との分界を書く。
2. 要求の群ごとの証拠対応表を作り、実行環境の列を実際の実行で埋める。
3. セキュリティ検証を子文書として、宣言的なファイルの検証を節として書く。
4. 可用性と復旧の試験をシステム受入れ設計へ足し、受入れと確認対象を書き下す。
5. サービス管理に当番、エスカレーション、宣言の水準、変更の分類、ポストモーテムを足す。
6. 保守に定期作業の表、試験の実施、廃止と引渡しの段階を足す。
7. 索引と参照元を追従させる。

## Tasks

- [x] T001 [Acceptance] 検証設計の索引から新設する子文書へのリンクを先に置き、`mise run check-links` が未作成のリンク先で失敗することを確認する。プロダクトの規範を変えないため、受け入れ RED の代替はこの検査である。
- [x] T002 [Design] 検証設計とテスト方針の分界を決め、担当範囲を宣言する。
- [x] T003 [Docs] 要求の群ごとの証拠対応表と、宣言的なファイルの検証の節を書く。
- [x] T004 [Docs] セキュリティ検証設計を書く。
- [x] T005 [Docs] システム受入れ設計に受入れと確認対象、可用性と復旧の試験を足す。
- [x] T006 [Docs] サービス管理を書き足す。
- [x] T007 [Docs] 保守に定期作業と試験の実施、廃止と引渡しを書き足す。
- [x] T008 [Docs] 検証と運用の索引、`docs/README.md` の文書体系を追従させる。
- [x] T009 [Verify] リンク、用語、SLO 参照、仕様、全体検証を通す。

## Verification

書いた層を、その層を見ている間に検査する。

| 変えたもの | 走らせる検査 |
| --- | --- |
| 文書間のリンクと見出し | `mise run check-links` |
| 採用語彙 | `mise run check-terminology` |
| この記録の frontmatter と完了記録 | `mise run check-work-items` |

最後に一度だけ走らせるもの。

- `mise run check-slo-references`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

要求と証拠の対応表を細かくすると、個々のテストの一覧になり、テストが増減するたびに古くなる。行は要求の群に対応させ、個々の対応はテスト側の `//spec:covers` に任せる。

検証設計へテストの書き方を書くと、[テスト方針](../../docs/development/testing.md)と二重になる。分界を冒頭で宣言し、越えたと気付いたら移す。

当番体制とエスカレーションは、実際の運用体制が無い状態で書くと架空の組織を記述することになる。役と判断基準だけを書き、人数、氏名、連絡先、具体的な時間は運用環境が決まってから埋める欄として残す。埋まっていない欄を、決まっているように見せない。

実行環境の列に「リポジトリ内」と書いた行が、実際には動かないことがある。そう書く前に、対応する `mise` タスクまたは CI ジョブが実在し、この環境で走ることを確かめる。

実行環境の列は、未実施の検証を「実環境が必要」と書くことで、環境さえあれば通るかのように読める。合否の判定が決まっていない行は「手段が未定」とし、環境の不足と設計の不足を混ぜない。

## Completion
- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の変更を挙げない。
  [検証設計](../../docs/verification/README.md)は、自分が持つ問いと[テスト方針](../../docs/development/testing.md)、運用文書、運用手順が持つ問いを冒頭の表で分け、要求の群ごとの証拠を、証拠の種類、実行、合否条件、実行環境の列で持つようになった。
  実行環境の列は進捗ではなく、証拠を得るために何を用意するかを「リポジトリ内」「実環境が必要」「手段が未定」で表す。
  宣言的なファイル（Kubernetes マニフェスト、Docker Compose 構成、監視資産、スキーマ定義、生成した参照文書）の検証は、独立した文書ではなく検証設計の節として置いた。デプロイ前にリポジトリ上で走らせる検査であり、稼働後を扱う運用文書には収まらず、実行のタイミングは[継続的インテグレーション](../../docs/development/continuous-integration.md)が既に正本として持つためである。
  [セキュリティ検証設計](../../docs/verification/security.md)は、脅威の状態ごとに求める証拠、制御の種類ごとの検証手段、許可しない操作の確認、見直しのタイミングを持つ。
  [システム受入れ設計](../../docs/verification/system-acceptance.md)は、受入れと、許可しないリクエストの確認対象を冒頭で書き下し、機能受入れの代表経路を領域ごとの表にし、可用性と復旧の試験として起こす障害、観測対象、成功条件、実行環境を持つ。
  [サービス管理](../../docs/operations/service-management.md)は、当番の役とエスカレーション、宣言の水準、変更の分類、ポストモーテムを持ち、error budget の扱いが未決であることの帰結を書いた。
  [保守](../../docs/operations/maintenance.md)は、定期作業を「頻度を決める条件」と担当の分界で持ち、可用性と復旧の試験の実施をその行に加え、依存更新の範囲と後退、廃止と引渡しの六段階を持つ。
  正本文書の集合を宣言する `tools/workspace/src/document-layout.ts` は、`docs/verification` に `security.md` を許すようになった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-links`
  - **Requirement**: N/A: 文書の担当範囲と証拠設計を書く変更であり、プロダクトの規範シナリオを変えない。
  - **Observed Failure**: 索引と作業項目から新設予定の子文書へリンクを置いた時点で、`docs/verification/README.md:17` と作業項目の 2 行について `relative link target does not exist` が 4 件失敗した。
  - **Detection Reason**: 検証設計から子文書へ到達できない状態と到達できる状態を、リンク先の実在で区別する。索引だけを書いて子文書を書かない誤りをこの検査が落とす。
  - **Retained**: 独立した文書をやめた運用検証について、`operational-verification.md` を削除した直後も同じ検査が参照元の 1 件を挙げ、参照を節へ向け直して解消した。
- **Unit RED Evidence**:
  - **Test**: `mise run check`（`check-repository` の正本文書集合）
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: 子文書を書いた直後、`docs/verification/security.md` と `docs/verification/operational-verification.md` について `not a canonical document; docs/verification/ holds only README.md, system-acceptance.md` が 2 件失敗した。
  - **Detection Reason**: 正本文書の集合は `tools/workspace/src/document-layout.ts` の宣言が持つ。宣言を更新せずにファイルだけを増やす誤りを、この検査が文書体系の外側にある文書として落とす。
- **Change-Resistance Results**:
  正本文書集合の宣言から子文書を外す故障注入は、実装前後の状態がそのまま相当する。宣言が無い状態で `mise run check` は所見を挙げて失敗し、宣言を足すと成功した。運用検証を独立文書からやめたときも、宣言から `operational-verification.md` を消し忘れていれば同じ検査が落ちる。文書の変更であり変異器の対象となる分岐を持たないため、`mise run test-go-mutation` は実行していない。
- **Verification Results**:
  - `mise run check-links` - passed
  - `mise run check-terminology` - passed
  - `mise run check-work-item-references` - passed
  - `mise run check-slo-references` - passed
  - `mise run check` - passed
  - `mise run test-tools` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 実行しない。文書と、文書体系の宣言だけの変更であり、ブラウザーへ到達する経路を変えない。
