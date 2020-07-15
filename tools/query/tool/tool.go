package tool

import (
	"github.com/hawell/redins/tools/query/query"
	"github.com/miekg/dns"
)

type QueryTool interface {
	Act(q query.Query, client *dns.Client)
	Result()
}
