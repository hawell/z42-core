package database

type User struct {
	Id int64
	Name string
}

type Zone struct {
	Id int64 `json:"-"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	Dnssec bool `json:"dnssec"`
	CNameFlattening bool `json:"cname_flattening"`
}

type Location struct {
	Id int64 `json:"-"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
}

type RecordSet struct {
	Id int64 `json:"-"`
	Type string `json:"type"`
	Value string `json:"value"`
	Enabled bool `json:"enabled"`
}

var SupportedTypes = []string{"a", "aaaa", "cname", "txt", "ns", "mx", "srv", "caa", "ptr", "tlsa", "ds", "aname"}
