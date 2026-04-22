import { GlassCard } from "./GlassCard";
import { TrendingUp, Users, Armchair, Activity } from "lucide-react";

interface StatsBarProps {
  free?: number;
  total?: number;
  reserved?: number;
  occupancy?: number;
}

export const StatsBar = ({ free, total, reserved, occupancy }: StatsBarProps) => {
  const fmt = (n?: number) => (n === undefined ? "—" : String(n));
  const pct = (n?: number) => (n === undefined ? "—" : `${Math.round(n * 100)}%`);
  const stats = [
    { icon: Armchair, label: "Свободно", value: fmt(free), total: total !== undefined ? ` / ${total}` : undefined },
    { icon: Users, label: "В резерве", value: fmt(reserved) },
    { icon: Activity, label: "Загрузка", value: pct(occupancy) },
    { icon: TrendingUp, label: "Этаж", value: total !== undefined ? `${total} мест` : "—" },
  ];
  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
      {stats.map((s) => (
        <GlassCard key={s.label} hover className="p-4 flex items-center gap-3">
          <div className="h-9 w-9 rounded-xl bg-foreground/[0.04] border border-border grid place-items-center">
            <s.icon className="h-4 w-4 text-muted-foreground" />
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{s.label}</p>
            <p className="font-display text-xl font-semibold">
              {s.value}
              {s.total && <span className="text-xs text-muted-foreground font-normal">{s.total}</span>}
            </p>
          </div>
        </GlassCard>
      ))}
    </div>
  );
};