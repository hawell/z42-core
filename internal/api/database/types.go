package database

import (
	"github.com/google/uuid"
)

type ObjectId string

const EmptyObjectId ObjectId = ""

func NewObjectId() ObjectId {
	return ObjectId(uuid.New().String())
}

type User struct {
	Id       ObjectId
	Email    string
	Password string
	Status   string
}

type NewUser struct {
	Email    string
	Password string
	Status   string
}

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
	UserStatusPending  = "pending"
)

const (
	VerificationTypeSignup = "signup"
)

type Zone struct {
	Id              ObjectId
	Name            string
	Enabled         bool
	Dnssec          bool
	CNameFlattening bool
}

type NewZone struct {
	Name            string
	Enabled         bool
	Dnssec          bool
	CNameFlattening bool
}

type ZoneUpdate struct {
	Enabled         bool
	Dnssec          bool
	CNameFlattening bool
	SOA string
}

type Location struct {
	Id      ObjectId
	Name    string
	Enabled bool
}

type NewLocation struct {
	Name    string
	Enabled bool
}

type LocationUpdate struct {
	Enabled bool
}

type RecordSet struct {
	Id      ObjectId
	Type    string
	Value   string
	Enabled bool
}

type NewRecordSet struct {
	Type    string
	Value   string
	Enabled bool
}

type RecordSetUpdate struct {
	Value   string
	Enabled bool
}

type ListItem struct {
	Id string `json:"id"`
	Enabled bool `json:"enabled"`
}

type List []ListItem
