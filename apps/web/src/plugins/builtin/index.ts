import { registerWidget } from "../registry";
import { TotalStatementsWidget } from "./TotalStatements";
import { ActiveUsersWidget } from "./ActiveUsers";
import { StatementsTodayWidget } from "./StatementsToday";
import { TrackedClientsWidget } from "./TrackedClients";
import { StatementTrendWidget } from "./StatementTrend";
import { CoverageWidget } from "./Coverage";
import { RecentStatementsWidget } from "./RecentStatements";

export function registerBuiltinWidgets() {
  registerWidget(TotalStatementsWidget);
  registerWidget(ActiveUsersWidget);
  registerWidget(StatementsTodayWidget);
  registerWidget(TrackedClientsWidget);
  registerWidget(StatementTrendWidget);
  registerWidget(CoverageWidget);
  registerWidget(RecentStatementsWidget);
}
