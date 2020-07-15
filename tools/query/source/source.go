package source

import "github.com/hawell/redins/tools/query/query"

type QueryGenerator interface {
	Init()
	Count() int
	GetQuery() query.Query
}
