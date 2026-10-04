# SET の受信

## 概要

この文書は、外部の送信側が受信のエンドポイントへ送る SET を検証し、失効エポックへ反映する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | SET の署名と発行者の検証、`jti` の再利用の拒否、主体の解決、失効エポックの前進 |
| 行為者 | System（外部の送信側からの `/ssf/streams/{stream_id}/events` への要求） |
| 扱わないもの | 受信側のストリームの登録は[SSF ストリームの管理](../stream/README.md)が扱う |

## モデル

受理した SET の `events` クレームから、失効を反映する対象のテナントの中のプリンシパルを決める。

| 主体の形式 | 解釈 |
| --- | --- |
| IdMagic 自身の送信側が使う独自の形式（`subject_type`、`tenant_id`、`principal_id`） | 名乗ったテナントとプリンシパル |
| RFC 9493 の `format=iss_sub` | `iss` が受信側のストリームの `trusted_issuer` と一致するときの `sub` |
| RFC 9493 の `format=opaque` | `id` |

形式は `format` のメンバーの有無で判別し、両方の形式を同時に満たす表現は受け付けない。
RFC 9493 の形式は自身のテナントを名乗らないので、テナントは受信側のストリームが属するテナントで決まる。
識別子は Agent の識別子として解決し、一致しなければ Agent に束縛した `OAuth2Client` の識別子として解決する。
どちらでも解決できない場合は拒否する。

| 拒否の結果（`verification_result`） | 条件 |
| --- | --- |
| `rejected_signature` | 署名を登録した鍵で検証できない、未知の鍵、改ざんを検知した |
| `rejected_replay` | 同じ `jti` の SET をすでに受理している |
| `rejected_subject_unresolved` | 主体をストリームのテナントの中で解決できない |

## 操作

### 外部の送信側による SET の受信

#### REQ-SHAREDSIGNALS-010 RFC 9493 の Subject Identifier で送られた SET も主体を解決する

- 受信側のストリームが SET を受理したとき、SharedSignals は、主体の Agent の失効エポックを進め、受理の記録を残し、202 を返し、`SecurityEventReceived` を発行する。
- `iss` が受信側のストリームの `trusted_issuer` と一致する間、`format=iss_sub` の主体を受けたとき、SharedSignals は、`sub` をストリームのテナントの識別子として解決する。
- `format=opaque` の主体を受けたとき、SharedSignals は、`id` をストリームのテナントの識別子として解決する。
- 主体の識別子を解決するとき、SharedSignals は、まず Agent の識別子として、次に Agent に束縛した `OAuth2Client` の識別子として探し、失効を Agent の識別子で記録する。
- `iss_sub` と `opaque` 以外の `format`、`trusted_issuer` と一致しない `iss`、どの Agent にも束縛先のクライアントにも一致しない識別子、ほかのテナントを名乗る主体を受けた場合、SharedSignals は、400 と `security_event_rejected` で拒否し、`verification_result=rejected_subject_unresolved` の `SecurityEventRejected` を発行し、失効を反映しない。
- 64 KiB を超える SET を受けた場合、SharedSignals は、413 と `security_event_token_too_large` で拒否する。
- SET の受信を拒否するとき、SharedSignals は、RFC 8935 の `err` と `description` を持つ本文を返す。
- **例**：EX-SHAREDSIGNALS-010-01、EX-SHAREDSIGNALS-010-02、EX-SHAREDSIGNALS-010-03、EX-SHAREDSIGNALS-010-04、EX-SHAREDSIGNALS-010-05、EX-SHAREDSIGNALS-010-06

#### REQ-SHAREDSIGNALS-003 署名が不正な SET は反映せずに拒否する

- 署名を登録した鍵で検証できないか、未知の鍵の SET か、改ざんした SET を受けた場合、SharedSignals は、400 と `security_event_rejected` で拒否し、`verification_result=rejected_signature` の `SecurityEventRejected` を発行し、失効エポックを変えない。
- 発行者が `trusted_issuer` と一致しないか、audience が受理する audience に含まれない SET を受けた場合、SharedSignals は、400 と `security_event_rejected` で拒否し、失効エポックを変えない。
- 存在しないストリーム、送信側のストリーム、受信側の設定のないストリームへ SET を受けた場合、SharedSignals は、400 と `security_event_rejected` で拒否し、失効エポックを変えない。
- **例**：EX-SHAREDSIGNALS-003-01

#### REQ-SHAREDSIGNALS-004 同じ jti の SET は一度だけ反映する

- 同じストリームで受理済みの `jti` の SET を受けた場合、SharedSignals は、400 と `security_event_rejected` で拒否し、`verification_result=rejected_replay` の `SecurityEventRejected` を発行し、失効エポックを変えない。
- **例**：EX-SHAREDSIGNALS-004-01

#### REQ-SHAREDSIGNALS-005 発行者が一致しても他テナントのストリームでは受理しない

- SET の主体を解決するとき、SharedSignals は、受信側のストリームが属するテナントの中だけを探し、ほかのテナントのプリンシパルに作用しない。
- **例**：EX-SHAREDSIGNALS-005-01

## セキュリティ上の考慮

受信のエンドポイントは、ブラウザーのセッションを持たない外部の送信側が呼ぶため、管理 API の認証の経路には載せない。
代わりに、SET の署名が受信側のストリームに登録した `trusted_issuer` の鍵で検証できること、`jti` が未使用であること、主体がそのストリームのテナントの中で解決できることを、すべて満たしたときにだけ受理する。
一つでも満たさなければ、失効を反映せずに拒否する。

受理した SET が変えられるのは、対象の Agent の失効エポックだけであり、それを進める以外の作用を持たない。
解決の探索の範囲は受信側のストリームが属するテナントを出ないので、ほかのテナントのプリンシパルを名乗る SET が届いても、そのテナントには作用しない。
