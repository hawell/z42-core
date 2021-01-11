package db

type User struct {
	Id int64
	Name string
}

type Zone struct {
	Id int64
	Name string
	Enabled bool
}

type Location struct {
	Id int64
	Name string
	Enabled bool
}

type RecordSet struct {
	Id int64
	Type string
	Value string
	Enabled bool
}