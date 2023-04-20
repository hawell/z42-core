package tool

import (
	"z42-core/tools/query/query"
	"github.com/miekg/dns"
)

type QueryTool interface {
	Act(q query.Query, client *dns.Client)
	Result()
}
