package database

import (
	"errors"
	"github.com/google/uuid"
	"github.com/hawell/z42/internal/types"
	jsoniter "github.com/json-iterator/go"
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
	VerificationTypeSignup  = "signup"
	VerificationTypeRecover = "recover"
)

type Zone struct {
	Id              ObjectId
	Name            string
	Enabled         bool
	Dnssec          bool
	CNameFlattening bool
	SOA             types.SOA_RRSet
	DS              string
}

type NewZone struct {
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	Dnssec          bool            `json:"dnssec"`
	CNameFlattening bool            `json:"cname_flattening"`
	SOA             types.SOA_RRSet `json:"soa"`
	Keys            types.ZoneKeys  `json:"keys"`
	NS              types.NS_RRSet  `json:"ns"`
}

type ZoneUpdate struct {
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	Dnssec          bool            `json:"dnssec"`
	CNameFlattening bool            `json:"cname_flattening"`
	SOA             types.SOA_RRSet `json:"soa"`
}

type ZoneDelete struct {
	Name string `json:"name"`
}

type Location struct {
	Id      ObjectId
	Name    string
	Enabled bool
}

type NewLocation struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
	Enabled  bool   `json:"enabled"`
}

type LocationUpdate struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
	Enabled  bool   `json:"enabled"`
}

type LocationDelete struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
}

type RecordSet struct {
	Id      ObjectId
	Type    string
	Value   types.RRSet
	Enabled bool
}

type NewRecordSet struct {
	ZoneName string      `json:"zone_name"`
	Location string      `json:"location"`
	Type     string      `json:"type"`
	Value    types.RRSet `json:"value"`
	Enabled  bool        `json:"enabled"`
}

func (r *NewRecordSet) UnmarshalJSON(data []byte) error {
	var dat struct {
		ZoneName string `json:"zone_name"`
		Location string `json:"location"`
		Type     string `json:"type"`
		Enabled  bool   `json:"enabled,default=true"`
	}
	if err := jsoniter.Unmarshal(data, &dat); err != nil {
		return err
	}
	value := types.TypeToRRSet[dat.Type]
	if value == nil {
		return errors.New("invalid record type")
	}
	val := struct {
		Value types.RRSet `json:"value"`
	}{
		Value: value(),
	}
	if err := jsoniter.Unmarshal(data, &val); err != nil {
		return err
	}
	r.ZoneName = dat.ZoneName
	r.Location = dat.Location
	r.Type = dat.Type
	r.Enabled = dat.Enabled
	r.Value = val.Value
	return nil
}

type RecordSetUpdate struct {
	ZoneName string      `json:"zone_name"`
	Location string      `json:"location"`
	Type     string      `json:"type"`
	Value    types.RRSet `json:"value"`
	Enabled  bool        `json:"enabled"`
}

func (r *RecordSetUpdate) UnmarshalJSON(data []byte) error {
	var dat struct {
		ZoneName string `json:"zone_name"`
		Location string `json:"location"`
		Type     string `json:"type"`
		Enabled  bool   `json:"enabled,default=true"`
	}
	if err := jsoniter.Unmarshal(data, &dat); err != nil {
		return err
	}
	value := types.TypeToRRSet[dat.Type]
	if value == nil {
		return errors.New("invalid record type")
	}
	val := struct {
		Value types.RRSet `json:"value"`
	}{
		Value: value(),
	}
	if err := jsoniter.Unmarshal(data, &val); err != nil {
		return err
	}
	r.ZoneName = dat.ZoneName
	r.Location = dat.Location
	r.Type = dat.Type
	r.Enabled = dat.Enabled
	r.Value = val.Value
	return nil
}

type RecordSetDelete struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

type ListItem struct {
	Id      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type List struct {
	Items []ListItem `json:"items"`
	Total int        `json:"total"`
}

func EmptyList() List {
	return List{
		Items: []ListItem{},
		Total: 0,
	}
}

type Event struct {
	Revision int
	ZoneId   string
	Type     EventType
	Value    string
}

type EventType string

const (
	AddZone        EventType = "add_zone"
	UpdateZone     EventType = "update_zone"
	DeleteZone     EventType = "delete_zone"
	AddLocation    EventType = "add_location"
	UpdateLocation EventType = "update_location"
	DeleteLocation EventType = "delete_location"
	AddRecord      EventType = "add_record"
	UpdateRecord   EventType = "update_record"
	DeleteRecord   EventType = "delete_record"
)
