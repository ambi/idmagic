# SAML の SSO の設計

この文書は、[SAML の SSO](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

クレームの発行と Assertion の署名には、WS-Federation と WS-Trust で共有している構築器と署名器（`backend/wsfederation/tokens_saml`）を再利用する。
これらは SAML のバージョン、Bearer の SubjectConfirmation、audience の制限をすでに扱っている。
この Context は署名の処理を作り直さず、`InResponseTo` の対応付けなど、SP 起点の流れに固有の入力だけを加える。

`goxmldsig` は、署名の対象の要素の末尾に Enveloped Signature を加える。
署名の後に要素を移動すると名前空間が組み直されてダイジェストの値が変わり、検証できなくなるので、署名した要素は移動しない。
この制約は、Assertion と Response のどちらに署名する場合にも当てはまる。

## データ

AuthnRequest の ID は、テナント、SP の entityID、要求の ID の組で、期限付きの予約として記録する。
予約は、記録がないか期限の切れた記録だけを上書きし、期限内の記録があれば再送として扱う。
期限の切れた予約は、掃除の処理が小さな単位で消す。

## 信頼性

予約の記録先に到達できない場合は、エラーを返し、Assertion を発行しない。
