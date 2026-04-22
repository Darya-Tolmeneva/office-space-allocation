import { GlassCard } from "./GlassCard";
import { LogOut } from "lucide-react";
import { motion } from "framer-motion";
import { useAuth } from "@/lib/auth/AuthContext";

export type View = "map" | "reservations" | "analytics";

const tabs: { id: View; label: string }[] = [
  { id: "map", label: "Карта" },
  { id: "reservations", label: "Резервации" },
  { id: "analytics", label: "Аналитика" },
];

interface HeaderProps {
  view: View;
  onViewChange: (v: View) => void;
}

export const Header = ({ view, onViewChange }: HeaderProps) => {
  const { user, logout } = useAuth();
  const initials = user?.fullName
    ? user.fullName.split(" ").map((p) => p[0]).slice(0, 2).join("").toUpperCase()
    : "?";
  return (
  <header className="sticky top-4 z-50 mx-4 lg:mx-8">
    <GlassCard variant="strong" className="flex items-center justify-between px-5 py-3">
      <div className="flex items-center gap-3">
        <div className="h-9 w-9 rounded-xl bg-gradient-to-br from-primary to-accent grid place-items-center shadow-sm">
          <span className="font-display text-sm font-bold text-primary-foreground">F</span>
        </div>
        <div>
          <h1 className="font-display text-base font-semibold leading-none">FlowDesk</h1>
          <p className="text-[10px] text-muted-foreground tracking-widest uppercase mt-1">Smart office</p>
        </div>
      </div>
      <nav className="hidden md:flex items-center gap-1 bg-muted/60 rounded-full p-1 relative">
        {tabs.map((t) => {
          const active = view === t.id;
          return (
            <button
              key={t.id}
              onClick={() => onViewChange(t.id)}
              className="relative px-4 py-1.5 text-sm rounded-full transition-colors"
            >
              {active && (
                <motion.span
                  layoutId="tab-pill"
                  className="absolute inset-0 bg-background shadow-sm rounded-full"
                  transition={{ type: "spring", stiffness: 380, damping: 32 }}
                />
              )}
              <span className={`relative z-10 ${active ? "text-foreground font-medium" : "text-muted-foreground hover:text-foreground"}`}>
                {t.label}
              </span>
            </button>
          );
        })}
      </nav>
      <div className="flex items-center gap-2">
        <div className="hidden sm:flex items-center gap-2 rounded-full bg-muted/60 pl-1 pr-3 py-1">
          <div className="h-7 w-7 rounded-full bg-gradient-to-br from-primary to-accent grid place-items-center text-[11px] font-semibold text-primary-foreground">
            {initials}
          </div>
          <span className="text-xs text-foreground/80 max-w-[140px] truncate">{user?.fullName}</span>
        </div>
        <button
          onClick={logout}
          aria-label="Выйти"
          className="rounded-full p-2 bg-muted/60 hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
        >
          <LogOut className="h-3.5 w-3.5" />
        </button>
      </div>
    </GlassCard>
  </header>
);
};