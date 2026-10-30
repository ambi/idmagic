---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-10-31
priority: p2
depends_on: [wi-286-outbound-event-hooks-and-audit-log-streaming]
change_kind: feature
affected_spec:
  - { path: docs/modules/sharedsignals/standards.md, requirement: SSF10-SUB-ID }
  - { path: docs/modules/sharedsignals/standards.md, requirement: SSF10-STATUS-NO-TRANSMIT }
  - { path: docs/modules/sharedsignals/features/stream.md, requirement: REQ-SHAREDSIGNALS-008 }
---

# 受信側が標準の手順で接続できるよう SSF 1.0 に適合する

## 動機

[[wi-286-outbound-event-hooks-and-audit-log-streaming]] で、テナントへの継続配送を独自 Webhook ではなく SSF に一本化した。
その理由は、受信側が SSF にだけ対応すれば済むことである。
しかし現在の送信側は管理者の手動登録を前提とし、SSF への適合は部分的なので、標準の受信側が標準の手順で接続できるとは限らず、一本化の利点を十分に得られない。

## 対象範囲

- 対象の受信製品を一つ選び、接続を確認する。
- その製品の接続に必要な SSF 1.0 の機能を実装する。候補は次のとおりである。

| 不足 | 内容 |
| --- | --- |
| Discovery | `/.well-known/ssf-configuration` がない |
| Stream Management API | 受信側がストリームを作成、更新、削除する API がない（仕様は SHOULD） |
| 受信側の認可 | 受信側をストリームと `aud` に結びつける仕組みがない |
| 検証イベント | Stream Verification がない（CAEP Interoperability Profile の draft では MUST） |
| 状態の扱い | `paused` がなく、無効化時の Stream Updated Event を送っていない |
| 主体の表現 | CAEP イベント内の `subject` が独自形式のまま |
| Poll（RFC 8936） | 未対応。必須ではないが、全ログ転送には向く可能性がある |

この表は wi-286 の調査によるもので、仕様の先頭 100,000 文字の要約に基づくため網羅的でない。

## 対象外

- 対象の製品の接続に不要な機能。適合の範囲を広げる項目を別に起票する。
- 独自 Webhook。SET を受けられない具体的な受信側が確認できたときに別の項目で検討する。
- SIEM 製品専用のコネクタと中継。

## 設計

着手時に次を解決し、ここへ記録する。どれも作るものを変えるので、実装の前に決める。

- 対象の受信製品。SSF の受信側の候補として SailPoint の製品と caep.dev の試験環境を確認している。実際の接続は未確認である。
- 受信側の認証方式と、管理者の登録との関係（管理者が発行した資格情報に結ぶか、受信側が直接登録するか）。
- `disabled` と `paused` の分担。自動無効化を `paused` へ移すかを含む。
- CAEP の `subject` を標準の形式へ変えるか。既存の受信処理（RFC 9493 の解釈）との整合を確かめる。

信頼境界と新しい主体の種類を加えるので、[脅威モデル](../../docs/design/security/threat-model.md)を見直す。
公開状態が `published` になった後は、Discovery の内容と API の形を戻せない。

## 計画

1. SSF 1.0 全文と CAEP Interoperability Profile を読み、不足の表を確定する。
2. 対象の受信製品を選び、現状で何が足りず接続できないかを確かめる。
3. 設計の問いを解決し、要件と TypeSpec を書く。
4. 必要な機能を実装する。
5. 対象の製品または試験環境と接続して確認する。

## タスク

- [ ] T001 [Research] SSF 1.0 全文と CAEP Interoperability Profile から不足を確定する。
- [ ] T002 [Research] 対象の受信製品を選び、現状での接続を試す。
- [ ] T003 [Spec] 設計の問いを解決し、要件、標準の宣言、TypeSpec、脅威モデルを更新する。
- [ ] T004 [App] T002 で必要と分かった機能を実装する。各機能のテストで RED を確認する。
- [ ] T005 [Acceptance] 対象の製品または試験環境との接続を確認する。
- [ ] T006 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 着手時に差分から選ぶ。少なくとも `check-spec`、`check-contract-drift`、`verify-changed` と、T005 の接続確認を含める。

## リスク

- 受信側が自分でストリームを作れるようになると、新しい信頼境界と主体が増える。認可の誤りは他テナントのイベントの漏えいにつながる。
- 対象の製品が要求する機能が想定より多い場合、この項目が大きくなる。T002 の結果で分割する。
- 対象の受信製品が接続できない、または SET を受ける受信側が見つからない場合は、SSF への一本化の判断を見直す。
