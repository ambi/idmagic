---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-10-01
priority: p2
depends_on: [wi-95161-declare-spec-impact-and-find-implicit-specifications]
change_kind: docs
affected_spec:
  - { path: docs/domain/authentication/federation/scenarios.feature.md, requirement: REQ-AUTHENTICATION-001 }
---

# Authentication の既存コードにある暗黙の仕様を書き起こす

## 動機

Authentication の実装には、仕様影響の宣言を導入する前から、コードにだけ存在する挙動がある。
エラー時のフォールバック、上限、順序、作用を起こさない条件などがコードにだけあると、実装を変えた人がそれを仕様変更だと認識できない。
仕様影響の宣言はこれからの変更だけを扱うので、既存の挙動は一度読んで分類する必要がある。

導入時点（2026-10-01）の `spec-review-candidates` の結果は次のとおりである。

| 項目 | 値 |
| --- | --- |
| 宣言済みの規則 | 37 件 |
| 仕様 ID を引くテストが実行しない本番コードの位置 | 796 箇所（50 パッケージ） |
| 対象にした仕様を引くテスト | 840 件 |

この数は読み始める位置の量を示すもので、仕様漏れの件数ではない。

機能ノードへの再編で見つかった入力を、この work item で分類する。

| 置き場所 | 値または条件 |
| --- | --- |
| `trusted-device/internals.md` | 絶対期限の上限 90 日、idle 期限 `last_used_at + min(30 日, max_age)` |
| `session/internals.md` | セッションの有効期限は作成時から固定 1 時間で延長しない、`last_seen_at` の更新は最短 5 分間隔、期限切れのレコードの保持 90 日 |
| `password/internals.md` | `min_length=12`、`max_length=128`、4 文字未満の識別子は照合しない、`history_depth=5`、`max_age_days` は 30〜3,650 日、リセットのトークンの `ttl=1800s` |
| `internals.md`（ルート） | ログインの流量制限、集計の閾値、保持期間、ステップアップの直近性 5 分、TOTP のパラメーター、WebAuthn の `challenge_bytes=32` と `timeout_seconds=120`、復旧コード 10 文字 10 個 |
| `security-notification/internals.md` | SMTP の待ち時間の上限 10 秒、既知の端末のレコードを `last_seen_at` から 365 日で消す |

文書と実装の食い違いも二件見つかっている。
`trusted-device/states.md` は「同じ理由での再失効は安全な no-op」と書くが、`TrustedDevice.Revoke` は失効済みなら理由を問わず何もせず、最初の理由を保つ。
`docs/design/application/api-guidelines.md` は不正な `limit` をエラーにすると定めるが、サインイン履歴の `parseLimitParam` は 0 以下または整数でない `limit` を黙ってデフォルト値にする。
どちらも現在の挙動を規則へ書き、要判断の欄に未決定の点を残す。

## 対象範囲

- コンテキストはすでに機能ノードへ再編されているので、書き起こした規則を該当する機能ノードへ書く。
- 次の候補を再取得し、パッケージ全体を[既存コードからの書き起こし](../docs/development/specification-first-workflow.md#既存コードからの書き起こし)の観点表で読む。
  - `mise run spec-review-candidates -- backend/authentication`
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

手順、分類、観点表は[仕様先行の開発ワークフロー](../docs/development/specification-first-workflow.md#既存コードからの書き起こし)が定める。
何を仕様として書くかは `SPECIFICATION_FORMAT.md` §6 の「仕様として書く実装上の細部」で判断し、規則は同じ節の「規則一件の書式」で書く。

着手時の `affected_spec` は起票時の仮の参照である。
着手時に、書き起こして宣言する規範要素へ置き換える。
新しい規則は `impact: modifies` として差分に現れる。

## 計画

1. 候補を再取得し、観点表でパッケージを読んで、細部を四種類に分類する。
2. 分類の一覧をこの work item に記録し、是正が必要な点は別の work item として起票する。
3. 維持すべき仕様と意図が疑わしい挙動を、機能ノードの規則として書く。
4. 書いた規則を引くテストを追加する。
5. 不要なコードを削除する。

未解決の問いはない。
個々の細部が仕様か実装詳細かは、着手後の分類で決める。

## タスク

- [ ] T001 [Plan] 候補を再取得し、観点表でパッケージを読んで、見つけた細部を四種類に分類した一覧をこの work item に記録する。
- [ ] T002 [Spec] 維持すべき仕様と意図が疑わしい挙動を、機能ノードの規則として書く。
- [ ] T003 [Acceptance] 書いた規則の例を引くテストを追加し、各テストが現在の挙動を固定することを、挙動を変える誤実装の注入で確認する。
- [ ] T004 [App] 不要と分類したコードを削除する。
- [ ] T005 [Plan] 是正が必要な点を、別の work item として起票する。
- [ ] T006 [Verify] 変更を検証する。

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
