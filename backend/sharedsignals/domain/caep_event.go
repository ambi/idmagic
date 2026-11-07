package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SsfSubject は CAEP イベントが指す対象。idmagic 自身が発行する access token の
// subject 表現 (tenant_id + principal_id) とそのまま対応させ、WorkloadIdentity の
// ような別テーブルでの subject mapping を新設しない。
type SsfSubject struct {
	SubjectType string // "Agent" または "User"
	TenantID    string
	PrincipalID string
}

const (
	SsfSubjectTypeAgent = "Agent"
	SsfSubjectTypeUser  = "User"
)

func (s SsfSubject) Validate() error {
	return spec.All(
		spec.Required("SubjectType", s.SubjectType, spec.OneOf(SsfSubjectTypeAgent, SsfSubjectTypeUser)),
		spec.Required("TenantID", s.TenantID),
		spec.Required("PrincipalID", s.PrincipalID),
	)
}

// InitiatingEntity は CaepEvent.InitiatingEntity の取りうる値。
type InitiatingEntity string

const (
	InitiatingEntityPolicy InitiatingEntity = "policy"
	InitiatingEntityAdmin  InitiatingEntity = "admin"
	InitiatingEntityUser   InitiatingEntity = "user"
)

func (e InitiatingEntity) Valid() bool {
	switch e {
	case InitiatingEntityPolicy, InitiatingEntityAdmin, InitiatingEntityUser:
		return true
	}
	return false
}

// CaepEvent は SET へ署名して包む前の、CAEP イベントの意味内容。
type CaepEvent struct {
	EventType        CaepEventType
	Subject          SsfSubject
	Reason           *RevocationReason
	EventTimestamp   time.Time
	InitiatingEntity InitiatingEntity
}

func (e CaepEvent) Validate() error {
	if err := spec.All(
		spec.Required("EventType", e.EventType, spec.Must(CaepEventType.Valid, "caep event type is not in enum")),
		spec.Required("EventTimestamp", e.EventTimestamp),
		spec.Required("InitiatingEntity", e.InitiatingEntity,
			spec.Must(InitiatingEntity.Valid, "initiating entity is not in enum")),
	); err != nil {
		return err
	}
	return e.Subject.Validate()
}
