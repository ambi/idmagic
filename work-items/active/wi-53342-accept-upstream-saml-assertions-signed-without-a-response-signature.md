---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-001 }
---

# 上流 SAML IdP が Assertion だけに署名した応答を受け付け、実物の IdP の記録で検証する

## 動機

Authentication の上流 SAML の接続は、応答の検証で Response の直下の `Signature` を必須にしている（`backend/authentication/internal/federation/protocol_saml/response.go` の `validateSignature`）。
SAML 2.0 Profiles 4.1.3.5 は、HTTP POST で届く Response の各 Assertion を署名で保護することを求め、その方法を個々の `<Assertion>` への署名と `<Response>` への署名のどちらでもよいとする。
Microsoft Entra ID の既定は Assertion だけへの署名である（Microsoft Learn「Advanced certificate signing options in a SAML token」、2026-11-28 に本文を読んだ。「Sign SAML assertion. This default option is set for most of the gallery applications.」）。
したがって、既定の設定の Entra ID を上流にした SAML のログインは、正しい応答でも署名がないとして拒否されるはずである。
AD FS と Okta も Assertion への署名を既定にすると理解しているが、本文は確かめていない。

この食い違いをテストが見つけられなかったのは、テストの応答が実装を書いた人の組み立てた XML だからである。
`protocol_saml/response_test.go` の `signedResponse` は、自分で作った鍵で Response 全体に署名した一形だけを作る。
外部へ要求を送る相手では、記録の再生に置き換えて既に複数の欠陥が見つかった（wi-65625、wi-27706 ほか）。
上流の SAML IdP は、要求を送った結果として応答を受け取る相手でありながら、この取り組みの洗い出し（wi-94480）が HTTP の送信先だけを見ていたため漏れた。

## 対象範囲

- 署名を Response と Assertion のどちらに置いた応答も受け付け、署名で保護された Assertion だけを属性の取り出しに使う。
- Response と Assertion の両方に署名した応答、Assertion だけに署名した応答、Response だけに署名した応答、どちらにも署名がない応答、署名のない Assertion を署名した Assertion と並べた応答を区別して検査する。
- 手元で起動できる SAML IdP（Keycloak の開発モードを第一の候補とする）から、署名の置き場所ごとの応答を記録し、`testing_httprecord` の形式で `protocol_saml/testdata/` に置いて再生する。
- 既存の手書きの `signedResponse` のテストのうち、記録の再生と同じ分岐を確かめるものを置き換え、記録にない分岐（改ざん、期限切れ、宛先の不一致）だけを残す。
- `REQ-AUTHENTICATION-001` の SAML の応答の検証の記述に、署名の置き場所の規則を書く。

## 対象外

- 暗号化した Assertion と IdP 起点の応答。[連携ログインの機能仕様](../../docs/modules/authentication/features/federation.md)が範囲外とする。
- SAML IdP として受け取る AuthnRequest の署名。[[wi-75603-verify-saml-redirect-signatures-over-the-received-encoding]] が扱う。
- Entra ID の実テナントでの確認。[[wi-79-entra-real-tenant-end-to-end-verification]] が扱う。

## 設計

### 署名の検証と取り出す要素

`goxmldsig` の `Validate` は、検証に成功した要素の写しを返す。
Response の署名を検証したときはその写しの中の Assertion を、Assertion の署名を検証したときはその写しを、属性の取り出しに使う。
元の木から Assertion を探し直すと、署名を包み替える攻撃（XML Signature Wrapping）で署名のない Assertion を読むことになるため、検証の結果だけを読む。
Assertion がちょうど一つであることは今と同じく求める。
Response の署名があり Assertion の署名もある場合は、両方を検証する。

### 記録の取り方

Keycloak は `start-dev` で起動でき、クライアントごとに Response と Assertion の署名を別々に切り替えられる。
バージョンを固定したコンテナを起動し、SP として AuthnRequest を送り、テスト用の利用者でログインして POST の応答を受け取る手順を、Vault の `record-transit` に倣ってスクリプトにするかを着手時に決める。
署名の形ごとに取り直す必要があるので、`mise` タスクにする見込みが高い。
記録には IdP の署名証明書（公開の値）と応答だけを残し、管理者の資格情報と鍵を残さないことを保存前に検査する。
応答は時刻と `InResponseTo` に縛られるので、再生では記録の時刻を入力として与え、要求の ID を記録と同じ値に固定する。

SimpleSAMLphp でも記録を取るかは、Keycloak の記録で署名の形が揃うかを見て決める。
Entra ID の応答の形は公開の文書に完全な例がなく、実テナントが要るので wi-79 に任せる。

### 採らない案

| 案 | 判断 |
| --- | --- |
| 手書きの `signedResponse` に Assertion だけに署名する形を足す | 採らない。署名の置き場所以外の実物の癖（名前空間の接頭辞、正規化の方式、`InclusiveNamespaces`、日時の小数秒）を確かめられない |
| Response の署名だけを求める現状を標準の逸脱として宣言する | 採らない。主要な IdP の既定の設定で接続できない |

## 計画

1. 一次資料（SAML Profiles 4.1.3.5、4.1.4.2、Core 5）の文面から期待値を書き、受け入れの RED を作る。
2. Keycloak の記録を取り、再生が今の実装で失敗することを確かめる。
3. 検証の結果だけを読む形に直し、署名を包み替えた入力を拒否することを確かめる。

## タスク

- [ ] T001 [Spec] `REQ-AUTHENTICATION-001` に署名の置き場所の規則を書く。
- [ ] T002 [Recording] Keycloak から署名の形ごとの応答を記録する手順と記録を置く。
- [ ] T003 [Acceptance] 記録の再生で、Assertion だけに署名した応答が拒否される RED を確かめる。
- [ ] T004 [Adapter] 検証した要素だけを読む形に直し、包み替えと署名のない Assertion の拒否を単体で検査する。
- [ ] T005 [Verify] 差分から選んだ検査を実行する。

## 検証

- 記録の署名値または Assertion の値を一文字変えると、再生が失敗する。
- 署名した Assertion の外に署名のない Assertion を足した入力を拒否する。
- `mise run test-go`（Authentication の連携の結合を含む）、`mise run lint-go`。

## リスク

署名の検証を緩める変更であり、誤ると署名のない Assertion を受け入れる。
検証の結果の写しだけを読むことと、包み替えた入力の拒否の検査で防ぐ。
