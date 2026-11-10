---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-11
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
---

# Entra Connect に合わせる ImmutableID の設定を通常の導入手順で理解できるようにする

## 動機

現在の「sourceAnchor 属性」という必須の自由入力欄は、AD 属性の選択、IdMagic の保存先の属性名、ユーザーごとの識別子の値を区別しにくい。
IdMagic の Connector は objectGUID を既定で取り込むが、新しい Entra Connect の標準的な構成は mS-DS-ConsistencyGuid を使う。
同期側の選択に合わせる必要と、その後の IdMagic 内部の属性参照を一つの欄で説明していることが導入の負担になっている。

Microsoft は[同期と外部 IdP の値の一致](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-design-concepts#impact-on-ad-fs-or-third-party-federation-configuration)を要求し、Okta は同等の選択を[Office 365 の属性マッピング](https://support.okta.com/help/s/article/How-to-change-the-ImmutableID-to-ms-consistency-guid-anchor-for-office365?language=en_US)で扱っている。

## 対象範囲

- 通常の Entra 設定では組み込み属性 on_premises_immutable_id を使い、内部の属性名の手入力を不要にする。
- 別の保存先を使う既存設定は維持し、読み取り専用の詳細と明示的な詳細設定から確認と変更ができるようにする。
- Entra Connect の選択を確認し、Connector の ldap.attributes.on_premises_immutable_id を合わせ、再同期して実値を照合する手順を表示する。
- 属性の欠損や不正値について対象ユーザーと直す箇所を示し、Graph で取得した onPremisesImmutableId と比較する手順を提供する。

## 対象外

- Connector の既定を無条件に変更すること、既存ユーザーの識別子を書き換えること、Entra Connect の設定変更。
- 別属性の選択で既存の一意性や利用者による編集禁止の検証を緩めること。
- IdMagic が Entra Connect の設定や利用者の ImmutableID を Graph から自動取得すること。

## 設計

AD の属性を選ぶ責任は Connector のマッピングに置き、Entra 連携の通常操作はその取り込み先を使う。
画面には「Entra のユーザー識別子（ImmutableID）」と、Entra Connect と同じ値が必要であることを示す。
管理者が Entra Connect で選んでいる属性を確認できない場合は推測で選ばず、確認手順を案内する。
新しい Connect でも既存構成の引き継ぎがあり、ConsistencyGuid が objectGUID と同じ場合もあるため、属性名だけで一致を断定しない。
GUID の生バイトの Base64 と、GUID の表示文字列や二重 Base64 を区別する。

API の既存の明示指定は維持し、既存の非標準マッピングを画面の初期値で上書きしない。
通常画面の単純化と、内部で値の取得元が必要なことを分けて扱う。
常に objectGUID に固定する案と、常に ConsistencyGuid に変える案は、同期側の既存設定を壊すため採らない。
汎用の属性マッピング機構や新たなモジュール間依存は追加しない。

## 計画

1. REQ-WSFEDERATION-001 に通常操作と詳細設定、既存設定の保持、値の確認の案内を追加する。
2. 既存の入力とエラー表示を、同期元、保存先、Entra の実値を区別する構成へ変更する。
3. objectGUID と ConsistencyGuid の両方について導入手順と画面の説明を揃える。

## タスク

- [ ] T001 [Spec] 通常操作の既定と詳細設定の保持、エラーの案内を定める。
- [ ] T002 [Acceptance] 内部の属性名を入力せず保存でき、既存の別属性の設定を編集しても保持する試験を先に置く。
- [ ] T003 [UI] 通常設定、詳細設定、欠損時の案内を実装する。
- [ ] T004 [Docs] Entra Connect の選択確認、Connector の対応付け、再同期と実値比較の手順を書く。
- [ ] T005 [Verify] 二つの AD 属性の構成と既存設定の互換性を確かめる。

## 検証

- `mise run test-ui-e2e-file -- frontend/tests/e2e/entra-source-anchor.spec.ts`：通常操作で内部キーの入力が不要であり、非標準の既存設定を黙って変更しないことと欠損時の案内を確かめる。
- `mise run test-go-package -- ./backend/wsfederation/...`：明示的な属性指定、一意性、編集禁止、欠損の既存の制約を保つことを確かめる。
- `mise run check-links` と `mise run verify-changed`：導入文書の参照と実装の影響範囲を検証する。

## リスク

自動選択したように見せて誤った識別子を使うと、対象ユーザーのサインインが失敗する。
Connector の既定値と既存ユーザーの値は変えず、同期側を確認して実値を比較するようにする。
Graph から取得する利用者の値は個人に紐付くため、比較用ファイルを監査ログや公開の試験結果へ残さない。
