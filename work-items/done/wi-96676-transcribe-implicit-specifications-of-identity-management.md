---
status: completed
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-10-01
priority: p2
depends_on: [wi-95161-declare-spec-impact-and-find-implicit-specifications]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 既存の挙動を規則として明示し、利用者が依存してよい境界値、デフォルト値、応答の形を初めて文書で約束する。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-96676-transcribe-implicit-specifications-of-identity-management.md }
initial_context:
  specification:
    - docs/domain/identity-management/user/scenarios.feature.md#REQ-IDMANAGEMENT-001
  typespec: []
  source:
    - docs/domain/identity-management
    - backend/idmanagement/domain
    - backend/idmanagement/usecases
    - backend/idmanagement/handlers_http
    - backend/idmanagement/user/domain
    - backend/idmanagement/user/usecases
    - backend/idmanagement/user/handlers_http
    - backend/idmanagement/group/domain
    - backend/idmanagement/group/usecases
    - backend/idmanagement/agent/domain
    - backend/idmanagement/agent/usecases
    - backend/idmanagement/agent/handlers_http
    - SPECIFICATION_FORMAT.md
  tests:
    - backend/idmanagement/handlers_http
    - backend/idmanagement/user/usecases
    - backend/idmanagement/agent/usecases
  stop_before_reading: [frontend, backend/oauth2, backend/authentication, spec/generated]
affected_spec:
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-033 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-034 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-035 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-036 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-037 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-038 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-039 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-040 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-041 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-042 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-043 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-044 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-045 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-046 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-047 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-048 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-049 }
  - { path: docs/domain/identity-management/user/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-050 }
  - { path: docs/domain/identity-management/account/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-051 }
  - { path: docs/domain/identity-management/account/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-052 }
  - { path: docs/domain/identity-management/account/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-053 }
  - { path: docs/domain/identity-management/account/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-054 }
  - { path: docs/domain/identity-management/user-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-055 }
  - { path: docs/domain/identity-management/user-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-056 }
  - { path: docs/domain/identity-management/user-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-057 }
  - { path: docs/domain/identity-management/user-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-058 }
  - { path: docs/domain/identity-management/user-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-059 }
  - { path: docs/domain/identity-management/group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-060 }
  - { path: docs/domain/identity-management/group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-061 }
  - { path: docs/domain/identity-management/group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-062 }
  - { path: docs/domain/identity-management/group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-063 }
  - { path: docs/domain/identity-management/group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-064 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-065 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-066 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-067 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-068 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-069 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-070 }
  - { path: docs/domain/identity-management/dynamic-group/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-071 }
  - { path: docs/domain/identity-management/group-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-072 }
  - { path: docs/domain/identity-management/agent/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/domain/identity-management/agent/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-074 }
  - { path: docs/domain/identity-management/agent/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-075 }
  - { path: docs/domain/identity-management/agent/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-076 }
  - { path: docs/domain/identity-management/agent/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-077 }
  - { path: docs/domain/identity-management/agent/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-078 }
---

# IdManagement の既存コードにある暗黙の仕様を書き起こす

## 動機

IdManagement の実装には、仕様影響の宣言を導入する前から、コードにだけ存在する挙動がある。
エラー時のフォールバック、上限、順序、作用を起こさない条件などがコードにだけあると、実装を変えた人がそれを仕様変更だと認識できない。
仕様影響の宣言はこれからの変更だけを扱うので、既存の挙動は一度読んで分類する必要がある。

導入時点（2026-10-01）の `spec-review-candidates` の結果は次のとおりである。

| 項目 | 値 |
| --- | --- |
| 宣言済みの規則 | 32 件 |
| 仕様 ID を引くテストが実行しない本番コードの位置 | 1072 箇所（22 パッケージ） |
| 対象にした仕様を引くテスト | 930 件 |

この数は読み始める位置の量を示すもので、仕様漏れの件数ではない。

## 対象範囲

- コンテキストの文書を機能ノードへ再編する。規範の意味を変えない構造変更として、書き起こしより先のコミットにする。
- 次の候補を再取得し、パッケージ全体を[既存コードからの書き起こし](../../docs/development/specification-first-workflow.md#既存コードからの書き起こし)の観点表で読む。
  - `mise run spec-review-candidates -- backend/idmanagement`
- 見つけた細部を、維持すべき仕様、意図が疑わしいが外部から依存され得る挙動、実装詳細、不要なコードの四種類に分類する。
- 維持すべき仕様は、現在の挙動を機能ノードの規則として書き、その規範 ID を引くテストを書く。
- 意図が疑わしい挙動は、現在の挙動を規則として書き、要判断の欄に未決定の点を残す。
- 不要なコードを削除する。

## 対象外

- 挙動の是正。
  現在の挙動を明示する変更と、挙動を修正する変更を分けるため、是正は見つけた点ごとに別の work item で扱う。
- 実装詳細の文書化。
  その細部だけが異なる二つの実装をどちらも正しいと判断できるなら、仕様へ書かない。
- 候補をゼロにすること。
  候補は着手点であり、実行されないコードが実装詳細の場合もある。
- `frontend/` の既存コードの探索。

## 設計

手順、分類、観点表は[仕様先行の開発ワークフロー](../../docs/development/specification-first-workflow.md#既存コードからの書き起こし)が定める。
何を仕様として書くかは `SPECIFICATION_FORMAT.md` §6 の「仕様として書く実装上の細部」で判断し、規則は同じ節の「規則一件の書式」で書く。

着手時の `affected_spec` は起票時の仮の参照である。
着手時に、書き起こして宣言する規範要素へ置き換える。
新しい規則は `impact: modifies` として差分に現れる。

### 機能ノード

コードの機能スライス `user`、`group`、`agent` に対応する機能ノードを置き、規則のまとまりが大きい `user` と `group` からは、セルフサービス、CSV、動的グループを別のノードに分けた。
分けたノードのコードは、それぞれ元のスライスにある。

| 機能ノード | 移した規則 | 節の語彙 |
| --- | --- | --- |
| `user` | REQ-IDMANAGEMENT-001、005、010〜013 と `UserLifecycle` | ライフサイクル |
| `account` | REQ-IDMANAGEMENT-002、003、016〜019 | API |
| `user-csv` | REQ-IDMANAGEMENT-004、006、007 | API |
| `group` | REQ-IDMANAGEMENT-015、024 | API |
| `dynamic-group` | REQ-IDMANAGEMENT-020〜023 と `DynamicMembershipEvaluationLifecycle` | API |
| `group-csv` | REQ-IDMANAGEMENT-008、026〜031 | API |
| `agent` | REQ-IDMANAGEMENT-009 と `AgentLifecycle` | ライフサイクル |

複数の機能にまたがる規則（REQ-IDMANAGEMENT-014、025、032）、User と Group が共有する CSV の基盤と `DataExportLifecycle`、管理 API の認可、予約ロール、CSV の共有の判断は、ルートに残した。

### 分類の結果

観点表で `backend/idmanagement` の全パッケージを読み、見つけた細部を次のとおり分類した。

| 細部 | 分類 | 反映先 |
| --- | --- | --- |
| ロールの空白の除去、重複の除去、並び、大文字と小文字の区別 | 維持すべき仕様 | REQ-IDMANAGEMENT-033 |
| CSV の見出しの BOM、完全一致の照合、秘密情報の 5 つの列、空のファイル | 維持すべき仕様 | REQ-IDMANAGEMENT-034 |
| 列の数が合わない行だけの拒否 | 維持すべき仕様 | REQ-IDMANAGEMENT-035 |
| 先頭のアポストロフィーの復号の条件 | 維持すべき仕様 | REQ-IDMANAGEMENT-036 |
| 転送ポリシーのデフォルト値と上限ちょうどの扱い | 維持すべき仕様 | REQ-IDMANAGEMENT-037 |
| 属性値のセルの正規の字句形 | 維持すべき仕様 | REQ-IDMANAGEMENT-038 |
| エクスポートの開始の列と絞り込みの検証 | 維持すべき仕様 | REQ-IDMANAGEMENT-039 |
| エクスポートの一覧の並びと直近 200 件の窓 | 維持すべき仕様、窓は意図が疑わしい | REQ-IDMANAGEMENT-040、wi-97149 |
| 終了したエクスポートの取り消しの 409 | 維持すべき仕様 | REQ-IDMANAGEMENT-041 |
| 作成のユーザー名の照合とパスワードポリシー、メールアドレスの重複の許容 | 維持すべき仕様、大小の区別と重複の許容は意図が疑わしい | REQ-IDMANAGEMENT-042、wi-97149 |
| JIT のメールアドレスの正規化と衝突、操作者、動的グループの未評価 | 維持すべき仕様、未評価は意図が疑わしい | REQ-IDMANAGEMENT-043、wi-97149 |
| 一覧の取得による期限切れの削除予約の完全削除 | 意図が疑わしい | REQ-IDMANAGEMENT-044、wi-97149 |
| 更新の `changed_fields`、何も変わらない更新の省略、`email_verified` の維持 | 維持すべき仕様、`email_verified` は意図が疑わしい | REQ-IDMANAGEMENT-045、wi-97149 |
| 無効化と再有効化の再実行、自分自身の無効化の拒否、記憶済みの端末の失効 | 維持すべき仕様 | REQ-IDMANAGEMENT-046 |
| 必須操作の付与と解除の再実行、未定義の値 | 維持すべき仕様 | REQ-IDMANAGEMENT-047 |
| 削除の予約の再実行、自分自身の拒否の順序、理由、下流への通知 | 維持すべき仕様 | REQ-IDMANAGEMENT-048 |
| 復元の猶予期間の境界、下流へ通知しないこと | 維持すべき仕様、通知しないことは意図が疑わしい | REQ-IDMANAGEMENT-049、wi-97149 |
| 完全削除の匿名化の値、消す記録、使用量、再実行 | 維持すべき仕様 | REQ-IDMANAGEMENT-050 |
| 本人のプロフィールの併合、拒否、`changed_fields` | 維持すべき仕様、`changed_fields` は意図が疑わしい | REQ-IDMANAGEMENT-051、wi-97149 |
| 本人へ開示する属性とデータのエクスポート | 維持すべき仕様、エクスポートの範囲は意図が疑わしい | REQ-IDMANAGEMENT-052、wi-97149 |
| メールアドレスの変更の起票の正規化、拒否、リンク、送信の失敗 | 維持すべき仕様、送信の失敗は意図が疑わしい | REQ-IDMANAGEMENT-053、wi-97149 |
| メールアドレスの変更の確定の作用と拒否のコード | 維持すべき仕様 | REQ-IDMANAGEMENT-054 |
| User の CSV の `attr:` と `custom:`、行の対象と重複、組み込み列のセル | 維持すべき仕様 | REQ-IDMANAGEMENT-055〜057 |
| CSV で作成する User のパスワードと必須操作 | 維持すべき仕様 | REQ-IDMANAGEMENT-058 |
| 所有を判定できないときの既存の User の行の拒否 | 維持すべき仕様 | REQ-IDMANAGEMENT-059 |
| Group の名前、説明、連絡先の正規化 | 維持すべき仕様、連絡先の扱いの差は意図が疑わしい | REQ-IDMANAGEMENT-060、wi-97149 |
| Group の属性とスキーマの有無、更新の省略 | 維持すべき仕様 | REQ-IDMANAGEMENT-061、062 |
| 手動のメンバーの追加の対象と再実行 | 維持すべき仕様 | REQ-IDMANAGEMENT-063 |
| Group の変更の下流への通知の失敗 | 意図が疑わしい | REQ-IDMANAGEMENT-064、wi-97149 |
| 動的グループの式の制約、版と有効化、一致の対象、無効化、古いジョブ、プレビュー | 維持すべき仕様 | REQ-IDMANAGEMENT-065〜069、071 |
| 動的な所属の変化のイベント | 意図が疑わしい | REQ-IDMANAGEMENT-070、wi-97149 |
| Group の CSV の名前の照合、連絡先と動的規則のセル | 維持すべき仕様 | REQ-IDMANAGEMENT-072 |
| Agent の登録の名前と所有者、束縛、更新、停止した Agent の操作 | 維持すべき仕様 | REQ-IDMANAGEMENT-073〜075、077 |
| Agent の無効化と再有効化の再実行、停止した Agent の削除 | 意図が疑わしい | REQ-IDMANAGEMENT-076、078、wi-97149 |
| 管理 API からの Group のエクスポートが `custom:` 列を拒否すること | 既存の規則に反する欠陥 | wi-47878 |
| 行数と項目の上限を超える CSV のインポートが、上限までの行を適用すること | 既存の規則に反する欠陥 | wi-95422 |
| 行数と項目の上限を超えるエクスポートが `export_failed` になること | 既存の規則に反する欠陥 | wi-20633 |
| `PendingDeletion` の User の無効化と再有効化、エクスポートの期限の基準 | 状態遷移表との食い違い | wi-18703 |
| 管理者による作成のパスワードポリシー違反の応答の形 | TypeSpec と API ガイドラインとの食い違い | wi-18773 |
| 通知の宛名の補完（`User.DisplayName`）、属性のロケールの読み出し | 実装詳細（通知の文面は Notification が持つ） | なし |
| 一覧の取得での件数の並行取得、`limit+1` による次ページの判定 | 実装詳細 | なし |
| インポートの計画のページの大きさ、エラーの成果物のページの大きさ | 実装詳細 | なし |
| `Agent` の束縛の競合時の再確認 | 実装詳細 | なし |
| 保存層や送信の失敗をそのまま返す分岐 | 実装詳細 | なし |

不要と判断したコードはなかった。
内部設計のうち規則と重なった値（CSV の転送ポリシーのデフォルト値）は規則へのリンクに置き換えた。
3 件の欠陥は、一時的なテストで実際の挙動を観測してから起票した。

## 計画

1. 候補を再取得し、観点表でパッケージを読んで、細部を四種類に分類する。
2. 分類の一覧をこの work item に記録し、是正が必要な点は別の work item として起票する。
3. 維持すべき仕様と意図が疑わしい挙動を、機能ノードの規則として書く。
4. 書いた規則を引くテストを追加する。
5. 不要なコードを削除する。

未解決の問いはない。
個々の細部が仕様か実装詳細かは、着手後の分類で決める。

## タスク

- [x] T001 [Plan] 候補を再取得し、観点表でパッケージを読んで、見つけた細部を四種類に分類した一覧をこの work item に記録する。コンテキストの文書を機能ノードへ再編する。
- [x] T002 [Spec] 維持すべき仕様と意図が疑わしい挙動を、機能ノードの規則として書く。
- [x] T003 [Acceptance] 書いた規則の例を引くテストを追加し、各テストが現在の挙動を固定することを、挙動を変える誤実装の注入で確認する。単体と受け入れの確認は `mise run test-go-test -- <package> <test>`、パッケージ単位は `mise run test-go-package -- <package>`、変異は `mise run test-go-mutation -- <package-directory>` で行う。
- [x] T004 [App] 不要と分類したコードを削除する。不要と分類したコードはなかった。
- [x] T005 [Plan] 是正が必要な点を、別の work item として起票する。
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run check`
- `mise run verify`

## リスク

- 現在の偶然の挙動を仕様として固定する危険がある。
  将来の実装が維持する義務を負うかを先に判断し、義務のない細部は実装詳細として仕様へ書かない。
- 被覆は、実行されているが仕様に書かれていない挙動を見つけない。
  候補だけに頼らず、観点表でパッケージ全体を読む。
- 割り当てた規範 ID は取り消せない。
  規則の単位を決めてから ID を割り当てる。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff` は、REQ-IDMANAGEMENT-033 から REQ-IDMANAGEMENT-078 までの 46 件の規則の追加を報告し、既存の規則の変更、削除、状態遷移の変更は報告しない。
  IdManagement の文書を、ユーザー、アカウントのセルフサービス、ユーザー CSV、グループ、動的グループ、グループ CSV、エージェントの 7 つの機能ノードへ分けた。
  この再編は規範の意味を変えない構造変更で、再編だけの時点で `spec-diff` は差分を報告しなかった。
  既存コードを観点表で読み、実装にだけあった挙動を 46 件の規則と 97 件の例として書き起こした。
  現在の挙動を規則として書き、振る舞いは変えていない。
  意図が疑わしい 14 点は規則に要判断として残し、wi-97149 で決める。
  既存の規則に反する欠陥 3 件（管理 API からの Group のエクスポートが `custom:` 列を拒否する、上限を超える CSV のインポートが上限までの行を適用する、上限を超えるエクスポートが `export_failed` になる）は、一時的なテストで挙動を観測してから wi-47878、wi-95422、wi-20633 として起票した。
  状態遷移表との食い違いは wi-18703、TypeSpec と API ガイドラインに反するパスワードポリシー違反の応答は wi-18773 として起票した。
  内部設計のうち規則と重なった CSV の転送ポリシーのデフォルト値は、規則へのリンクに置き換えた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（規範 ID をテストが引いているかの検査）
  - **Requirement**: REQ-IDMANAGEMENT-033
  - **Observed Failure**: 規則を書いた直後、97 件の例すべてについて `EX-IDMANAGEMENT-0NN-MM is declared, but no test names it.` で失敗した。規則の書式（担保手段のシンボル、節、曖昧な語）の検査は、その時点で通っていた。
  - **Detection Reason**: 書き起こしは既存の挙動を固定する作業なので、実装より先に失敗するのは、宣言した例をどのテストも引いていないという検査である。挙動そのものを固定できていることは、下の変異と誤実装の注入で確かめた。
- **Unit RED Evidence**:
  - **Test**: `backend/idmanagement/domain/csv_rules_test.go` の `TestCSVByteLimitAcceptsTheLimitItself`、`TestCSVWriterCountsBytesAcrossFlushes`、`backend/idmanagement/group/usecases/group_rules_test.go` の `TestStaleReconcileJobLeavesMembershipUntouched`
  - **Requirement**: REQ-IDMANAGEMENT-037
  - **Observed Failure**: N/A: 既存の挙動を固定するテストなので、書いた時点で通過した。代わりに、変異テストで生き残った境界の変異（成果物の大きさの比較 `>` から `>=`、書き出しの残りの勘定 `max-written` から `max+written`、古い版の判定 `!=` から `==`）を、追加したテストが検出することを変異テストの再実行で観測した。
  - **Detection Reason**: 規則が定める境界（上限ちょうどは受け付ける、複数回の書き出しにまたがって勘定する、古い版のジョブは所属を変えない）を、どれか一つでも外した実装はテストが拒否する。
- **Change-Resistance Results**:
  `mise run test-go-mutation` を、変更した 5 つのパッケージに実行した。
  このツールは各パッケージ自身のテストだけを実行するので、HTTP の入口から固定した規則の変異は生き残る。

  | パッケージ | 検出 | 生き残り |
  | --- | --- | --- |
  | `backend/idmanagement/domain` | 50 / 58 | 8 |
  | `backend/idmanagement/usecases` | 46 / 55 | 9 |
  | `backend/idmanagement/agent/usecases` | 79 / 82 | 3 |
  | `backend/idmanagement/user/usecases` | 321 / 357 | 36 |
  | `backend/idmanagement/group/usecases` | 338 / 369 | 31 |

  生き残った変異は次のとおり分類した。

  | 生き残った変異 | 分類 |
  | --- | --- |
  | 転送ポリシーの値自体の検証、エラーの列名と行番号 | この作業の規則の外 |
  | エクスポートの保持期限の算術、期限切れの表示 | この作業の規則の外。期限の基準は wi-18703 が扱う |
  | エクスポートの開始で User の方言の有無を見る分岐 | 等価な変異。方言がない分岐の検証も同じ入力を拒否する。Group の方言の欠落は wi-47878 が扱う |
  | Agent の束縛の競合時の再確認 | 逐次のテストでは到達しない並行時の分岐 |
  | 保存層、送信、通知のエラーをそのまま返す分岐 | 等価な変異。メモリーの保存層は失敗しない |
  | `account_profile.go` の変更の判定 3 件 | HTTP のテストが固定する。変わらない名前を変更として扱う誤実装を手で注入し、`TestAccountProfileRecordsAttributesEvenWhenTheValueIsUnchanged` が失敗した |
  | インポートのジョブ、エラーの成果物のページ、計画のページの大きさ | 既存の規則（REQ-IDMANAGEMENT-004、026）の領域で、この作業の規則の外 |
  | 規則の無効化の件数、版が変わったときの入れ替えの件数 | 無効化の件数はイベントに載らない。入れ替えの件数（`Removed` の加算）は `DynamicMembershipEvaluated` に載るがテストが読んでいない。テストの不足として残す |
  | 容量だけを変える算術 | 等価な変異 |

  ツールが表せない配線の誤りは手で注入した。
  ユーザー一覧の取得から期限切れの完全削除を外す変更、本人のデータのエクスポートへすべての属性を載せる変更、メンバーの追加で `Active` でない User を拒否する変更、JIT で動的グループを評価する変更を注入し、どれも対応するテストが失敗した。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-work-items` - 成功
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功（サンドボックスの外で実行）
