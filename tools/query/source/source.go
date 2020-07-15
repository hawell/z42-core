package source

import "github.com/hawell/z42/tools/query/query"

type QueryGenerator interface {
	Init()
	Count() int
	GetQuery() query.Query
}
