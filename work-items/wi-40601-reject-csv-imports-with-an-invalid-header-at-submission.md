---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-06
priority: p2
depends_on: [wi-95422-reject-csv-imports-that-exceed-the-row-or-field-limit]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004, impact: conforms }
  - { path: docs/modules/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-055, impact: conforms }
  - { path: docs/modules/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-026, impact: conforms }
  - { path: docs/modules/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-029, impact: conforms }
---

# 見出しが不正な CSV のインポートを、投入の時点で拒否する

## 動機

EX-IDMANAGEMENT-004-03、EX-IDMANAGEMENT-026-03、EX-IDMANAGEMENT-029-03 は、見出しに未知の列、重複した列、`password` または `password_hash` を含む CSV について、「インポートの投入は拒否される」と定める。
EX-IDMANAGEMENT-029-04 は、見出しに `membership_state` のないメンバーシップの CSV を、ファイル全体として `invalid_header` で拒否すると定める。
REQ-IDMANAGEMENT-055 は、組み込みの拡張属性を `custom:<key>` の列で指定した User の CSV を `invalid_header` で拒否すると定める。

実装は、これらの見出しの誤りを投入の時点で検査しない。
投入は成功してプレビューのジョブを作り、ジョブが見出しを読んだ時点で `invalid_header` を一件のエラーとして記録して、成功として終わる。
wi-95422 では、この振る舞いを `TestCharacterize{User,Group,GroupMembership}ImportPreviewOfMalformedFiles` の「禁止した見出し」の事例として固定し、判断を後に回した。

利用者には、ファイルが拒否されたのではなく、行が 0 件で誤りが 1 件のプレビューが成功したように見える。
見出しの誤りでは行を一つも計画しないので、プレビューを適用しても User も Group もメンバーシップも変わらない。
それでも、成功したプレビューは適用でき、利用者は投入をやり直す代わりに適用を試せてしまう。

## 対象範囲

- User、Group、メンバーシップの CSV のインポートで、見出しの誤りをプレビューの投入の時点で検査し、400 と `invalid_header` で拒否する。プレビューのジョブも成果物も作らない。
- 対象にする見出しの誤りは、プレビューのジョブが今 `invalid_header` で記録するものすべてである：未知の列、重複した列、禁止した列（`password`、`password_hash` ほか）、テナントのスキーマにない `custom:<key>` の列、組み込みの拡張属性を指す `custom:<key>` の列、メンバーシップの `membership_state` の欠落。
- wi-95422 で判断を後に回した特性化テストのうち、「禁止した見出し」の事例を、要件を名指すテストへ置き換える。

## 対象外

- CSV の構文の誤り（`invalid_csv`）の扱い。途中の行に構文の誤りがあるファイルでは、今もその前の行が計画される。投入で拒否するかは別に判断する。
- 投入の後にテナントの属性スキーマが変わった場合の扱い。プレビューと適用のジョブは今と同じく見出しを読み直し、`invalid_header` を一件のエラーとして記録する。どの行も計画しないので、User も Group も変わらない。
- REQ-IDMANAGEMENT-026 の本文は、スキーマにない `custom:<key>` の列を「行を `rejected` にする」と書く。一方、EX-IDMANAGEMENT-026-10 と実装は、ファイル全体を `invalid_header` で拒否する。この食い違いの解消は、この作業に含めない。

## 設計

### 見出しの検査を投入と計画器で共有する

種別ごとに受理する列は、今は計画器の中で決めている。

| 種別 | 受理する列の決め方 | 計画器の中の追加の検査 |
| --- | --- | --- |
| User | `userdomain.NewUserCSVSchema(EffectiveUserAttributeDefs)` の `Accepts` | なし |
| Group | テナントの Group 属性スキーマから作る `Accepts` | なし |
| メンバーシップ | `groupdomain.NewGroupMembershipCSVSchema()` の `Accepts` | `MissingRequiredColumn` で `membership_state` の欠落を拒否する |

投入は wi-95422 で、`idmdomain.CopyCSVWithinPolicy` が入力を一度だけ読み、上限を検査しながら成果物ストアへ書く形になった。
この関数に見出しの検査を渡せるようにし、投入と計画器が同じ種別ごとの検査を使う。

| 操作 | シグネチャの案 | 役割 |
| --- | --- | --- |
| 見出しの検査 | `func(header []string) *CSVError` を種別ごとに作る | 受理する列の判定、禁止した列、重複、必須の列をまとめて判定する |
| 上限内の複製 | `CopyCSVWithinPolicy(output, input, policy, checkHeader)` | 見出しを読んだ時点で `checkHeader` を呼び、誤りなら成果物を作らずに返す |

User と Group の投入は、テナントの属性スキーマを読む必要がある。
`UserImportStartDeps` と `GroupImportStartDeps` に、計画器と同じスキーマを読むポートを加え、組み立ての境界で配線する。
HTTP の層は、`*CSVError` をすでに 400 と安定コードへ変換している。画面も投入の 400 と `invalid_header` の文言を持つので、変えない見込みである。

### 未解決の問い

- 禁止した列と重複した列の検査は、`NewCSVReader` の中にある。種別ごとの検査へ移すか、`newCSVReader` に検査の関数を渡す形にするかは、着手時に決める。どちらでも規則の実装は一つに保つ。

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| プレビューのジョブを `invalid_header` で失敗させる | 失敗したジョブの公開表示はエラーコードを運ばないので、画面は「インポート処理が失敗しました」としか示せない。具体例は投入の拒否を求める |
| 投入で成果物を保存した後に読み直して見出しを検査する | 拒否したファイルの成果物がストアに残る |

## 計画

1. 三つのインポートで、見出しの誤りを持つファイルの投入が拒否され、ジョブも成果物も作られないことのテストを書き、RED を確かめる。
2. 見出しの検査を投入と計画器で共有し、投入で拒否する。
3. 「禁止した見出し」の特性化テストの事例を、要件を名指すテストへ置き換える。

## タスク

- [ ] T001 [Acceptance] 三つのインポートの、見出しの誤りを持つファイルの投入のテストを書き、RED を確かめる。
- [ ] T002 [App] 見出しの検査を投入と計画器で共有し、投入で `invalid_header` を返す。
- [ ] T003 [Test] 特性化テストの「禁止した見出し」の事例を、要件を名指すテストへ置き換える。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`
- `mise run test-go-mutation` を、変更した usecases のパッケージと `backend/idmanagement/domain` にかける
- `mise run verify`
- `mise run test-ui-e2e`

## リスク

- User と Group の投入が、テナントの属性スキーマを読むようになる。スキーマを読めないときは投入を失敗させ、見出しを検査しないまま受け付けない。
- 投入の時点と、ジョブが見出しを読む時点でスキーマが異なると、判定が分かれる。どちらの結果でも行は一つも計画されない。
