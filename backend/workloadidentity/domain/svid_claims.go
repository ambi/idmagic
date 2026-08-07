package domain

import (
	"errors"
	"time"
)

// WorkloadSVIDClaims は検証を通過した外部 attestation token の claim
// (backend/workloadidentity/verification_jose アダプタが tokens_jose の検証結果を
// この形へ変換する。usecases 層は tokens_jose を直接知らない)。
type WorkloadSVIDClaims struct {
	Issuer    string
	Subject   string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// SVIDVerificationError の Reason は VerifyWorkloadAttestation が
// WorkloadAttestationRejected の reason にそのまま使う。
var (
	ErrSVIDMalformed        = errors.New("workload_svid: malformed token")
	ErrSVIDInvalidSignature = errors.New("workload_svid: signature invalid")
	ErrSVIDIssuerMismatch   = errors.New("workload_svid: issuer mismatch")
	ErrSVIDAudienceMismatch = errors.New("workload_svid: audience mismatch")
	ErrSVIDExpired          = errors.New("workload_svid: expired")
	ErrSVIDTTLExceeded      = errors.New("workload_svid: ttl exceeded")
	ErrSVIDKeysUnavailable  = errors.New("workload_svid: signing keys unavailable")
)
