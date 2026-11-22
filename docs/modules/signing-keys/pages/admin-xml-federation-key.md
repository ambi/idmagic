# XML フェデレーションの署名鍵

この文書は、XML フェデレーションの署名鍵の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[SigningKeys](../README.md) / [ページ一覧](README.md) / XML フェデレーションの署名鍵

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [鍵](../../../design/application/pages/admin/keys/README.md)

`admin-xml-federation-key`　`/admin/keys/xml-federation`

信頼先の更新を確認して共有 XML 署名鍵を切り替える

| 項目 | 内容 |
| --- | --- |
| 主担当 | [SigningKeys](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 署名鍵または Entra フェデレーション |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 現行、準備中と検証用の鍵、Entra、WS-Federation と SAML の信頼先、証明書と手順のコピー、信頼先ごとと全体の更新確認、切り替え直前の再照合と準備の取消確認 |

利用手順は[SigningKeys のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| prepare | ready | 次の鍵を準備 | 常時 | IdMagic.SigningKeys.Operations.RotateTenantSigningKey | 準備した鍵と信頼先を再取得する | [XML フェデレーションの署名鍵](admin-xml-federation-key.md) / prepared | 対象の識別子と操作の入力値 |
| activate | prepared | 切り替えを確認 | 全信頼先の更新確認があり再取得した内容が一致 | IdMagic.SigningKeys.Operations.RotateTenantSigningKey | 準備した鍵を署名に使い、表示を再取得する | [XML フェデレーションの署名鍵](admin-xml-federation-key.md) / ready | 対象の識別子と操作の入力値 |
| cancel | prepared | 準備の取消を確認 | 常時 | IdMagic.SigningKeys.Operations.DisableTenantKey | 準備した鍵を無効にして再取得する | [XML フェデレーションの署名鍵](admin-xml-federation-key.md) / ready | 対象の識別子と操作の入力値 |
| entra | ready | Entra の設定 | 常時 | UI 内の操作 | 移動先を表示する | [Entra フェデレーション](../../ws-federation/pages/admin-entra-federation.md) / ready | なし |
| domain | ready | 信頼先のドメインを開く | 常時 | UI 内の操作 | 移動先を表示する | [Entra ドメインの詳細](../../ws-federation/pages/admin-entra-federation-domain.md) / ready | ドメイン |
| back | ready | 署名鍵へ戻る | 常時 | UI 内の操作 | 移動先を表示する | [署名鍵](admin-keys.md) / ready | なし |

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
| prepared | 次の鍵と各信頼先の更新確認を表示する。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/signing-keys/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[SigningKeysのページ一覧へ戻る](README.md)
