package domain

import (
	"time"

	z "github.com/Oudwins/zog"

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

var ssfSubjectSchema = z.Struct(z.Shape{
	"SubjectType": z.String().OneOf([]string{SsfSubjectTypeAgent, SsfSubjectTypeUser}).Required(),
	"TenantID":    z.String().Min(1).Required(),
	"PrincipalID": z.String().Min(1).Required(),
})

func (s SsfSubject) Validate() error {
	return spec.Validate(ssfSubjectSchema, &s)
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

var caepEventSchema = z.Struct(z.Shape{
	"EventType": z.StringLike[CaepEventType]().TestFunc(
		func(value *CaepEventType, _ z.Ctx) bool { return value.Valid() },
		z.Message("caep event type is not in enum"),
	).Required(),
	"EventTimestamp": z.Time().Required(),
	"InitiatingEntity": z.StringLike[InitiatingEntity]().TestFunc(
		func(value *InitiatingEntity, _ z.Ctx) bool { return value.Valid() },
		z.Message("initiating entity is not in enum"),
	).Required(),
})

func (e CaepEvent) Validate() error {
	if err := spec.Validate(caepEventSchema, &e); err != nil {
		return err
	}
	return e.Subject.Validate()
}
