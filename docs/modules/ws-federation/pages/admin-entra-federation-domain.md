# Entra ドメインの詳細

この文書は、Entra ドメインの詳細の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[WsFederation](../README.md) / [ページ一覧](README.md) / Entra ドメインの詳細

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [フェデレーション](../../../design/application/pages/admin/federation/README.md)

`admin-entra-federation-domain`　`/admin/federation/entra/$domain`

ドメインの設定と外部で観測した信頼情報を照合する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [WsFederation](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | フェデレーション一覧 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 設定と署名鍵の適用状況、コマンドとメタデータのコピー、観測 JSON の貼り付け・ファイル読込、登録解除の確認と移行手順への案内 |

利用手順は[WsFederation のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| record | ready | 観測結果を記録 | 常時 | IdMagic.WsFederation.Operations.RecordEntraFederationObservation | 信頼情報の照合結果を更新する | [Entra ドメインの詳細](admin-entra-federation-domain.md) / ready | 対象の識別子と操作の入力値 |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [Entra ドメインを編集](admin-entra-federation-edit.md) / ready | なし |
| remove | ready | 登録解除を確認 | 解除が成功 | IdMagic.WsFederation.Operations.RemoveEntraFederationDomain | 一覧へ戻る | [Entra フェデレーション](admin-entra-federation.md) / ready | なし |
| keys | ready | 署名鍵を管理 | 常時 | UI 内の操作 | 移動先を表示する | [XML フェデレーションの署名鍵](../../signing-keys/pages/admin-xml-federation-key.md) / ready | なし |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [Entra フェデレーション](admin-entra-federation.md) / ready | なし |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |
| invalid | 入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。 |
| submitting | 実行中の操作を示し、二重送信を防ぐ。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/ws-federation/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[WsFederationのページ一覧へ戻る](README.md)
