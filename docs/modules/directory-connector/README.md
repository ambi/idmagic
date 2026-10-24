# DirectoryConnector

## 責務と境界

利用者の閉域に置く実行単位 `idmagic-directory-connector` として、Active Directory と OpenLDAP を読み、Sourcing が公開する Connector API へ User の差分を送る。
IdMagic のサーバーとは別に配布され、別々に版が進む。
このモジュールとサーバーの間の契約は、TypeSpec の Connector API（`/directory-connector/v1/...`）だけである。

| 扱わないもの | 担当 |
| --- | --- |
| Connector の登録、呼び出しの認証、項目の適用、取り込みの実行の記録 | `Sourcing` |
| User の記録の正 | `IdManagement` |

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 登録 | 運用者の `idmagic-directory-connector enroll` | 利用者の閉域のホスト | [運用者による登録](features/runtime.md#運用者による登録) |
| 同期 | `idmagic-directory-connector run` の間隔 | 利用者の閉域のホスト | [同期](features/runtime.md#同期) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| サーバーのモジュールを import せず、Connector API のワイヤの型を TypeSpec に合わせて自分で持つ | 配布済みの Connector とサーバーは版がずれる。サーバーの内部の型を共有すると、サーバーの内部の変更が配布済みの Connector との互換性を気づかずに壊す |
| LDAP の方言の差、属性の対応付け、差分の位置をこのモジュールに閉じる | サーバーには正規化した項目だけが見え、方言を足してもサーバーを変えない |
| IdMagic の `bootstrap` を使わず、自分の設定ファイルだけを読む | IdMagic の運用者ではなく利用者が配置し、IdMagic の構成を持たない |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [Directory Connector の実行環境](features/runtime.md) | 公開鍵の登録、LDAP の読み取り、User の項目への対応付け、差分の送信と再試行。導入と運用の手順は同じ文書の実現方式から参照する |
