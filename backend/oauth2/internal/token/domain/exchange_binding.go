package domain

// ExchangeProof は Token Exchange の要求が示した鍵の証明である。ハンドラーが検証した
// DPoP Proof の JWK サムプリントと、mTLS の証明書のサムプリントを持つ。
type ExchangeProof struct {
	JKT     string
	X5TS256 string
}

// DecideExchangeBinding は交換で発行するトークンの送信者制約を決める (REQ-OAUTH2-056)。
// 第 2 戻り値が false なら交換を拒否する。
//
// 送信者制約付きの subject token は、同じ方式の同じ鍵の証明があるときだけ交換でき、
// 発行するトークンは同じ制約を引き継ぐ。クライアント認証と今回の Proof は要求者の鍵を
// 示すだけで、元のトークンの鍵を示さない。照合しなければ、盗んだ束縛済みトークンを
// 別の鍵か Bearer へ書き換えられる。
func DecideExchangeBinding(subject *SenderConstraint, proof ExchangeProof) (*SenderConstraint, bool) {
	if subject == nil {
		switch {
		case proof.JKT != "":
			return &SenderConstraint{Type: SenderConstraintDPoP, JKT: proof.JKT}, true
		case proof.X5TS256 != "":
			return &SenderConstraint{Type: SenderConstraintMTLS, X5TS256: proof.X5TS256}, true
		}
		return nil, true
	}
	// 鍵の空の制約を空の証明と一致させない (fail-closed)。
	switch subject.Type {
	case SenderConstraintDPoP:
		if subject.JKT != "" && subject.JKT == proof.JKT {
			return &SenderConstraint{Type: SenderConstraintDPoP, JKT: subject.JKT}, true
		}
	case SenderConstraintMTLS:
		if subject.X5TS256 != "" && subject.X5TS256 == proof.X5TS256 {
			return &SenderConstraint{Type: SenderConstraintMTLS, X5TS256: subject.X5TS256}, true
		}
	}
	return nil, false
}

// ReboundExchangeBinding は交換の許可があるときの送信者制約を決める (REQ-OAUTH2-056)。
// 許可は元の鍵の証明に代わる根拠で、Bearer への書き換えの根拠にはならないので、証明を必須にする。
func ReboundExchangeBinding(proof ExchangeProof) (*SenderConstraint, bool) {
	switch {
	case proof.JKT != "":
		return &SenderConstraint{Type: SenderConstraintDPoP, JKT: proof.JKT}, true
	case proof.X5TS256 != "":
		return &SenderConstraint{Type: SenderConstraintMTLS, X5TS256: proof.X5TS256}, true
	}
	return nil, false
}
