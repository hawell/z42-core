package database

type User struct {
	Id       int64
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
	Id              int64
	Name            string
	Enabled         bool
	Dnssec          bool
	CNameFlattening bool
}

type Location struct {
	Id      int64
	Name    string
	Enabled bool
}

type RecordSet struct {
	Id      int64
	Type    string
	Value   string
	Enabled bool
}

var SupportedTypes = []string{"a", "aaaa", "cname", "txt", "ns", "mx", "srv", "caa", "ptr", "tlsa", "ds", "aname"}
