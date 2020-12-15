package types

import "time"

type HealthCheckItem struct {
	Protocol  string    `json:"protocol,omitempty"`
	Uri       string    `json:"uri,omitempty"`
	Port      int       `json:"port,omitempty"`
	Status    int       `json:"status,omitempty"`
	LastCheck time.Time `json:"lastcheck,omitempty"`
	Timeout   int       `json:"timeout,omitempty"`
	UpCount   int       `json:"up_count,omitempty"`
	DownCount int       `json:"down_count,omitempty"`
	Enable    bool      `json:"enable,omitempty"`
	DomainId  string    `json:"domain_uuid,omitempty"`
	Host      string    `json:"host,omitempty"`
	Ip        string    `json:"ip,omitempty"`
	Error     error     `json:"-"`
}
