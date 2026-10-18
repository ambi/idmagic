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
  reason: 監査検索が使う username の解決と委譲モードの定数の参照先だけを変える。検索対象のテナント、未解決時の空の結果、照会エラー、選択肢の応答、永続状態とイベントを維持する。
---

# 監査検索から User の保存型と OAuth2 の実装への逆向きの参照をなくす

## 動機

wi-21934 の循環のうち、Audit から IdManagement へは username の解決、Audit から OAuth2 へは選択肢に使う委譲モードの定数だけが必要である。
監査の書き込みを利用するモジュールへ Audit が戻る参照を切り、監査の検索を保存型と導出ロジックの変更から隔離する。

## 対象範囲

- Audit が `UsernameLookup(context.Context, tenantID, username string) (userID string, found bool, err error)` の関数型を公開する。
- 既存の HTTP の組み立て地点が UserRepository からこの関数へ変換し、未配線は nil のままにする。
- TypeSpec の `DelegationMode` の値を、既存の Go 仕様バインディング `backend/shared/spec/enums.go` に置く。
- OAuth2 の既存の型名と定数名を型別名と定数参照で保ち、導出規則は OAuth2 に残す。
- 境界の負債、依存図と担当モジュールの現在の設計を同期する。

## 対象外

- 検索の認可、範囲、絞り込み、照会のタイミングの変更。
- トークンや監査から委譲モードを導く計算の変更。
- User の型や Repository、委譲の計算を共有ライブラリへ移すこと。
- 既存テストの変更。

## 設計

入力は REQ-AUDIT-001、REQ-AUDIT-006、REQ-AUDIT-007、REQ-OAUTH2-049、`parseAuditEventQuery`、`DelegationMode` と、正式な HTTP の経路を通る監査検索の既存テストである。
SQL、確定の境界、イベントと通知を変えず、もとの一回の `FindByUsername` を同じ引数で呼ぶ。
解決するテナントは監査の経路の側で決め、全テナントの検索でも username の解決は操作主体のテナントだけに行う。
空の User ID と存在しない User を区別するため、戻り値に `found` を持たせる。

| 案 | D2・D4・D5・D6 との適合 | User または委譲の表現が変わった場合の波及 | 採否 |
| --- | --- | --- | --- |
| 公開パッケージから UserRepository と OAuth2 の型を参照し続ける | 非公開参照はないが、要求していない保存操作を見せ、循環を残す | 監査が提供側の保存型と実装の変更に追従する | 不採用 |
| Audit の関数型へ組み立て地点で変換する | D4 と D6 に従い、必要な ID と存在の有無だけを渡す | 変換だけが User の表現に追従する | 採用 |
| 関数型の代わりに Repository を写した公開型を増やす | 読み取りの1操作に不要な型と保存操作を増やす | 提供側の構造への依存が残る | 不採用 |
| 委譲モードのワイヤ語彙を既存の仕様バインディングへ置く | `ClientType` や `GrantType` と同じ TypeSpec の型の対応だけを共有する。D5 が禁じる業務の計算と保存型の共通化は行わない | ワイヤ語彙は一か所、導出の規則は OAuth2 に閉じる | 採用 |
| 選択肢の値を起動時に注入する | 固定の列挙に省略・誤配線の余地を加える | 配線とテスト前提にも追従が必要になる | 不採用 |

新しい公開パッケージとモジュールを作らず、既存の Audit の ports と仕様バインディングだけを使う。
未確認の製品上の判断は残さず、変更するのは現在の語彙の定義場所と表現の変換だけである。

## 計画

1. 仕様バインディングと照会の関数型を定義する。
2. 組み立て地点で既存の UserRepository を変換し、Audit の参照を移す。
3. 不要になった負債を境界検査から求め、依存図と設計を同期する。

## タスク

- [x] T001 [Design] 依存の向きと変換の担当を確定する。
- [x] T002 [App] 型と参照を移し、正式な HTTP の経路を結ぶ。
- [x] T003 [Docs] 負債と現在の設計を同期する。
- [x] T004 [Verify] 既存の検索と委譲の検査を変更せずに通す。

## 検証

- `mise run report-contract-evidence -- --changed` で HTTP の組み立てへの影響と参照不足を確認する。
- `mise run test-go-changed -- main` で検索、委譲と逆依存の既存テストを race 付きで実行する。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet -- main` で循環の解消と新しい違反がないことを検査する。
- `mise run lint-go`、`mise run check-work-items`、`mise run check-links` で実装と記録を検査する。
- `mise run spec-diff` と `mise run check-spec-impact -- main` で規範仕様の変更がないことを確かめる。
- テストを変更しない構造変更として、振る舞いを保つ変更の証拠を使う。

## リスク

username を解決できない場合に絞り込みを外すと、要求した対象以外の監査を返してしまう。
nil の照会、見つからない User、照会のエラーを既存どおり区別し、既存の検索のテストを変えない。

## 要約

`spec-diff` は規範仕様の差分なしを示した。
監査検索の username の解決を Audit の関数型へ変え、User の保存型からの変換を既存の HTTP の組み立て地点へ閉じた。
TypeSpec の委譲モードの値を既存の Go 仕様バインディングへ集め、OAuth2 の型名と定数名、導出規則を保った。
検索のテナント境界、照会のエラーと未解決時の空の結果、選択肢の応答、永続状態とイベントは変えていない。
Audit を含む循環4件を除き、先行する2項目と合わせて循環の負債を53件から45件へ減らした。
既存テストは変更せず、残る循環の担当と作用の判断、および今回一括して置き換えない理由は wi-21934 に記録した。
