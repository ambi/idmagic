# Saml Internals

## SSO Profile scope

初期対応の範囲は SAML 2.0 の Web Browser SSO Profile に限る。HTTP-Redirect (deflate と Base64) および HTTP-POST (Base64) のバインディング、署名済みの Response と Assertion、メタデータの公開、SP 起点と IdP 起点の SSO、Single Logout を提供する。SAML ECP、暗号化された Assertion、IdMagic が SAML SP として外部 IdP と連携する機能は対象外とし、必要になった時点で別の実装単位として扱う。対応範囲を狭めることで、SAML で知られている署名ラッピング攻撃への露出を抑える。

クレームの発行と Assertion の署名には、WS-Federation と WS-Trust で共有している構築器と署名器 (`backend/wsfederation/tokens_saml`) を再利用する。これらは SAML のバージョン、Bearer SubjectConfirmation、audience の制限をすでに扱っている。この Context では署名処理を作り直さず、`InResponseTo` の対応付けなど SP 起点のフローに固有の入力だけを追加する。

デフォルトでは Assertion に署名し、Response への署名は任意に有効化できる。これは Okta や Entra が提供する「Sign Response」に相当する。`goxmldsig` は、署名対象要素の末尾に Enveloped Signature を追加する。署名後に要素を移動すると名前空間が再構成されてダイジェスト値が変わり、検証できなくなるため、署名済み要素は移動しない。この制約は Assertion と Response のどちらに署名する場合にも適用する。

相互運用時の安全性を保つ検証はドメイン層に集約し、フェイルクローズで処理する。Issuer は登録済み SP の entityID と完全に一致しなければならない。`AssertionConsumerServiceURL` は SP の許可リストと照合して任意の宛先への転送を防ぎ、audience は SP の entityID に限定する。検証結果を確定できない場合や値が一致しない場合は、すべて拒否する。

## Identity provider profiles

各サービスプロバイダーは、テナント内の IdP プロファイル 1 つだけに関連付ける。テナントに必ず存在し変更できない `default` プロファイルは複数のサービスプロバイダーで共有し、短い `/saml/*` ルートを使用する。追加のプロファイルは `/saml/idp/{profile_id}/*` ルートとプロファイル固有の entityID を使用する。`shared` プロファイルは複数のサービスプロバイダーで共有できるが、`dedicated` プロファイルを割り当てられるサービスプロバイダーは最大 1 つとする。どちらも同じモデルで表し、プロトコル、永続化、管理のすべての経路で同じ信頼境界の規則を適用する。

プロファイル管理 API は、サーバーが生成した正式な entityID と、メタデータ、SSO、SLO、証明書取得用の各 URL、証明書のフィンガープリントを返す。関連付けられているサービスプロバイダーの数も返し、UI では使用中のプロファイルを削除できないようにする。最終的な整合性は Repository で保証し、デフォルトプロファイルの変更、`dedicated` プロファイルへの複数サービスプロバイダーの割り当て、使用中のプロファイルの削除はいずれも拒否する。

SSO と SLO では、リクエスト先のルートからプロファイルを特定し、対象サービスプロバイダーに関連付けられたプロファイルと一致することを確認する。Destination の検証には、そのプロファイルの正式なエンドポイントを使用する。これらを組み合わせて検証することで、ある信頼境界に対する正当なリクエストが、同じテナントの別のプロファイルを介して再送されることを防ぐ。

## Tenant signing

すべてのリクエストで、発行の直前にテナントとプロファイルに対応する署名器を取得する。署名プロバイダーは、そのスコープで有効な `XmlFederationSigning` 資格情報を選ぶ。署名プロバイダー、秘密鍵を扱う署名器、X.509 証明書のいずれかを利用できない場合は、フェイルクローズで失敗させる。リクエストごとに署名器を取得することで、プロセス内で共有された証明書がテナントやプロファイルの境界を越えて使われることを防ぐ。

各プロファイルのメタデータでは、有効な証明書に加え、同じ鍵スコープに属する有効期限内の検証用証明書をすべて公開する。新しい Assertion と Response の署名には現在有効な資格情報だけを使用するが、検証用証明書を重複して公開することで、サービスプロバイダーはローテーション直前に発行されたメッセージも検証できる。XML の構文解析と正規化には、XML 署名用に選定した検証済みライブラリを使用する。

## AuthnRequest replay recording

`saml_authnrequest_replays` は、AuthnRequest の ID を初めて受信したときだけ記録する。`RecordIfNew` は `INSERT ... ON CONFLICT DO NOTHING` を実行し、挿入された行数によって初回のリクエストか再送かを判定する。
