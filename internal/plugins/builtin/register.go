package builtin

import "github.com/SGoerzen/LRSolid/internal/plugins"

func Register(r *plugins.Registry) {
	r.Register(TotalStatements{})
	r.Register(ActiveUsers{})
	r.Register(StatementsToday{})
	r.Register(TrackedClients{})
	r.Register(StatementTrend{})
	r.Register(Coverage{})
	r.Register(RecentStatements{})
}
