---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: User と Group の詳細に表示する SCIM 由来の照会を要求側のポートへ移す。200 応答の scim_source、対象テナント、照会失敗時の省略、永続状態とイベントを維持する。
---

# User と Group の SCIM 由来の表示を IdManagement の照会ポートから取得する

## 動機

`module-cycle:IdManagement->Sourcing` は、User と Group の詳細で SCIM 由来を表示するために、IdManagement が Sourcing の Repository 全体を受け取ることで生じる。
Sourcing は User の操作に IdManagement を使うため、表示の照会だけの逆向きの依存を切る。
wi-21934 の再計測から、SCIM の書き込みの原子性を変えずに独立して取り込める変更として切り出す。

## 対象範囲

- IdManagement が `SCIMUserLookup.IsSCIMUser(context.Context, tenantID, userID string) (bool, error)` を公開する。
- Group についても、公開 `group/ports` に `SCIMGroupLookup.IsSCIMGroup(context.Context, tenantID, groupID string) (bool, error)` を置き、同じ方針で既存の GroupRef 照会へ結ぶ。
- Sourcing の既存の `source_idmanagement` アダプターが、既存の単体の UserRef 照会からこの値へ変換する。
- 正式な HTTP の組み立てからポートを渡し、User の詳細は Sourcing の保存型を参照しない。
- 解消した負債と依存図、モジュールの現在の設計を更新する。

## 対象外

- SCIM の書き込み経路、UserRef の保存形式、クォータ、監査、原子性の変更。wi-11577 が扱う。
- 表示の照会が失敗したとき、User の詳細取得自体を失敗させる変更。
- バッチの所有者ガードを表示の照会へ流用すること。
- 既存テストの構築と期待結果の変更。

## 設計

入力は `HandleGetAdminUser`、`ScimRepository.FindUserRefByUserID`、`source_idmanagement.UserOwnershipGuard` と、両モジュールの責務表である。
対象は読取りだけであり、User と UserRef の保存、トランザクション、イベントの順序は変わらない。
対象の User・Group とそのテナントは認可後に取得した記録から取り、値が見つかった場合だけ従来どおり `scim_source: "SCIM"` を返す。
Repository が未配線なら、公開する照会も nil にする。
照会がエラーを返しても、既存ハンドラーと同じくエラーを応答へ伝えず、得られた有無を使う。

| 案 | D2・D4・D6 との適合 | 由来の表示を変える場合の波及 | 採否 |
| --- | --- | --- | --- |
| Sourcing の Repository 全体を受け取る | 保存や Group の操作まで見せ、循環が残る | 表示側が保存型の変更に追従する | 不採用 |
| IdManagement の照会ポートへ Sourcing が適合する | 必要な有無だけを公開し、既存のアダプターと組み立て地点を使う | Sourcing の保存変更はそのアダプターに閉じる | 採用 |
| 汎用の所有者ガードで1件を照会する | ガードは source-managed 全般の拒否用であり、SCIM と決めて表示する意味とは異なる | 新しい取り込み元を足すと誤ったラベルを返しうる | 不採用 |
| 共有ライブラリへ保存型を移す | SCIM の対応と User の意味を要し D5 に適合しない | 共有基盤へ業務の型が広がる | 不採用 |

新しいモジュールとパッケージを増やさず、既存の公開 `user/ports` と `group/ports` に必要最小限の照会を置く。
残る仮定は、現在の表示契約が SCIM の有無だけを示すことである。
取り込み元を増やして由来の表示を一般化する場合は、その機能の仕様から表現を選び直す。

## 計画

1. ポートと変換を既存の配置に追加する。
2. 正式な経路を結び、IdManagement から Sourcing の import を除く。
3. 負債台帳と依存図を同期し、既存の User、SCIM、正式な HTTP 経路のテストを実行する。

## タスク

- [x] T001 [Design] 境界と表示の契約を確定する。
- [x] T002 [App] ポートと変換を追加し、正式な経路を結ぶ。
- [x] T003 [Docs] 負債台帳と現在の設計を同期する。
- [x] T004 [Verify] テストを変更せず、影響する既存の検査を通す。

## 検証

- `mise run check-work-items` と `mise run check-links` で記録の形式と参照を検査する。
- `mise run report-contract-evidence -- --changed` で構成入口からの影響と操作の参照を確認する。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet -- main` で循環の解消と新規違反がないことを検査する。
- `mise run test-go-changed -- main` で既存の User、SCIM と HTTP 経路のテストを race 付きで実行する。
- `mise run lint-go`、`mise run check-spec-impact -- main`、`mise run spec-diff` で静的規則と規範への非影響を確かめる。
- テストを変えない構造変更として、振る舞いを保つ変更の証拠を使う。

## リスク

nil の Repository をインターフェースで包むと、未配線を照会できる値に見せてしまう。
Sourcing の公開する組み立て操作が、Repository の有無を確認してからアダプターを返す。

## 要約

`spec-diff` は規範仕様の差分なしを示した。
User と Group の詳細の SCIM 由来を、IdManagement が公開する読取りだけのポートへ移し、Sourcing の既存の対応照会を正式な HTTP の経路へ結んだ。
対象のテナント、由来の表示、未配線と照会失敗時の扱い、永続状態とイベントは維持し、既存テストを変更していない。
IdManagement から Sourcing への逆向きの参照をなくし、Sourcing を含む循環の3件を負債台帳から除いた。
先行する wi-61185 と合わせて循環の負債は53件から49件になった。
SCIM の書き込みの原子性と監査・クォータの是正は wi-11577 の対象として残る。
