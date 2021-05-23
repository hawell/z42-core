package database

import "github.com/google/uuid"

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

type Location struct {
	Id      ObjectId
	Name    string
	Enabled bool
}

type RecordSet struct {
	Id      ObjectId
	Type    string
	Value   string
	Enabled bool
}

var SupportedTypes = []string{"a", "aaaa", "cname", "txt", "ns", "mx", "srv", "caa", "ptr", "tlsa", "ds", "aname"}
