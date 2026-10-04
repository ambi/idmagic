# RP と Entra フェデレーションの管理の設計

この文書は、[RP と Entra フェデレーションの管理](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

GUID の形をした sourceAnchor の値は、ImmutableID として使う前に、.NET の `Guid.ToByteArray()` のバイト順で base64 に符号化する。
これは AD FS と Entra の慣行である。
すでに base64 の値は、そのまま通す。

## 信頼性

バイト順を誤ると、Entra は Assertion を社内の同じユーザーへ関連付けられず、アカウントの重複やサインインの失敗を招く。
Entra 側では原因を特定しにくいので、サインインの失敗が続く場合は、まず発行した ImmutableID と、社内のディレクトリが同期した値を突き合わせる。

Hybrid Azure AD Join の端末の登録（`windowstransport` とコンピューターアカウントの Kerberos）は範囲の外である。
設定の案内では、managed や PHS、あるいは AD FS を併存させる配置へ誘導する。
