package source

import "z42-core/tools/query/query"

type QueryGenerator interface {
	Init()
	Count() int
	GetQuery() query.Query
}
