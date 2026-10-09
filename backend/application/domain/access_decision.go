package domain

// ApplicationAccessDecision は、フェデレーションの開始を Application の割り当てとサインイン
// ポリシーで判定した結果である。OAuth2、Saml、WsFederation の開始経路が読む。
type ApplicationAccessDecision struct {
	Allowed        bool
	StepUpRequired bool
	ApplicationID  string
	Reason         string
	// TrustedDeviceAllowed は、実効ポリシーが記憶済みの信頼済みデバイスによる MFA の
	// 充足を認めるかどうか (wi-91)。StepUpRequired のときだけ意味を持ち、false なら
	// 呼び出し側は cookie を見ずに本物の第二要素を要求する。
	TrustedDeviceAllowed bool
}
