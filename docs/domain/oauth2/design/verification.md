# OAuth2 の検証

この文書は、OAuth2 の要件と統制のうち、例のテストのほかに確かめているものを扱う。
システム全体の検証の方式は[検証設計](../../../design/verification/README.md)に従う。

## パースの境界のファズテスト

攻撃者が値を決められる入力の解析と照合は Go native fuzzing の対象である。

`RedirectURIAllowed` は `redirect_uri` と `post_logout_redirect_uri` の唯一の照合規則である。受理するのは登録済みのいずれかとバイト単位で完全一致するときに限り、大文字小文字の同一視もパーセントエンコーディングやパスの正規化も行わない。いずれを許しても、攻撃者は登録済み URI を接頭辞に持つ別の宛先へ認可コードを配送できる。表明は厳密性と非空虚性の対で置く。厳密性だけでは「常に拒否する」実装も通ってしまうからである。

`VerifyPKCES256` は S256 の往復で表明する。正しく導出した challenge は受理し、verifier と challenge のどちらを 1 文字変えても拒否する。`NormalizeUserCode` は冪等であり、宣言済みの英数字だけを残す。とりこぼせば同じ user_code が照合できず、広く畳めば総当たりの空間が縮む。

クライアント認証の主体を決める経路も対象である。`ParseClientCertificateHeader` は URL エスケープ、PEM、base64 DER の 3 通りの包み方を受け取るので、どの経路で届いても同じ証明書は同じ thumbprint にならなければならない。`VerifyClientAssertion` と `VerifyDPoPForResource` は、こちらが署名していない値を必ず拒否する（`alg: none` と対称鍵混同を corpus に含む）。正当な値が受理されることは、それぞれの表駆動テストが別に押さえる。`ValidateJWKSURI` と `IsClientIDMetadataDocumentURL` は、受理した文字列を解析し直したときに https でホストを持ち userinfo も fragment も持たないことを表明する。生文字列に対する接頭辞判定へ退行すると破れる。
