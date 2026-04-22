import { GlassCard } from "./GlassCard";
import { Desk } from "./FloorPlan";
import { Button } from "./ui/button";
import { Calendar, Clock, MapPin, Sparkles, Zap, X, Loader2 } from "lucide-react";
import { useState } from "react";

interface Props {
  desk: Desk | null;
  onReserve: (id: string) => void | Promise<void>;
  onAutoPick: () => void | Promise<void>;
  reserving?: boolean;
  autoPicking?: boolean;
}

export const BookingPanel = ({ desk, onReserve, onAutoPick, reserving, autoPicking }: Props) => {
  const [time, setTime] = useState("full");

  return (
    <GlassCard variant="strong" className="p-6 flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">Резервирование</p>
          <h2 className="font-display text-2xl font-bold mt-1">Выбор места</h2>
        </div>
        <div className="glass rounded-xl px-3 py-2 text-xs flex items-center gap-2">
          <Calendar className="h-3.5 w-3.5 text-primary" />
          <span>Сегодня</span>
        </div>
      </div>

      {/* Auto pick CTA */}
      <button
        onClick={() => onAutoPick()}
        disabled={autoPicking}
        className="group rounded-2xl glass glass-hover px-5 py-4 flex items-center justify-between border border-border disabled:opacity-60"
      >
        <div className="flex items-center gap-3">
          <div className="h-9 w-9 rounded-xl bg-primary/15 border border-primary/30 grid place-items-center">
            {autoPicking ? <Loader2 className="h-4 w-4 text-primary animate-spin" /> : <Sparkles className="h-4 w-4 text-primary" />}
          </div>
          <div className="text-left">
            <p className="font-medium text-sm">Подобрать автоматически</p>
            <p className="text-xs text-muted-foreground">Сервер подберёт первое подходящее</p>
          </div>
        </div>
        <Zap className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
      </button>

      <div className="h-px bg-gradient-to-r from-transparent via-border to-transparent" />

      {/* Selected desk */}
      {desk ? (
        <div className="space-y-4 animate-fade-in">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-xs text-muted-foreground">Выбранное место</p>
              <p className="font-display text-3xl font-semibold text-foreground mt-0.5">{desk.label}</p>
            </div>
            <div className="glass rounded-full p-2">
              <MapPin className="h-4 w-4 text-muted-foreground" />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-2">
            {[
              { v: "full", l: "Весь день", t: "09–18" },
              { v: "am", l: "Утро", t: "09–13" },
              { v: "pm", l: "Вечер", t: "13–18" },
            ].map((opt) => (
              <button
                key={opt.v}
                onClick={() => setTime(opt.v)}
                className={`rounded-2xl p-3 border transition-all ${
                  time === opt.v
                    ? "border-primary bg-primary/10 text-foreground"
                    : "border-border/50 text-muted-foreground hover:border-border"
                }`}
              >
                <Clock className="h-3.5 w-3.5 mb-1.5 mx-auto" />
                <p className="text-xs font-medium">{opt.l}</p>
                <p className="text-[10px] opacity-70">{opt.t}</p>
              </button>
            ))}
          </div>

          <Button
            onClick={() => onReserve(desk.id)}
            disabled={reserving}
            className="w-full h-11 rounded-xl bg-primary hover:bg-primary/90 text-primary-foreground font-medium border-0"
          >
            {reserving ? (
              <span className="flex items-center gap-2"><Loader2 className="h-4 w-4 animate-spin" /> Бронируем…</span>
            ) : (
              <>Забронировать {desk.label}</>
            )}
          </Button>
        </div>
      ) : (
        <div className="text-center py-8 space-y-2">
          <div className="h-16 w-16 rounded-3xl glass mx-auto grid place-items-center animate-float">
            <MapPin className="h-7 w-7 text-muted-foreground" />
          </div>
          <p className="text-sm text-muted-foreground">Выберите место на карте<br />или используйте авто-подбор</p>
        </div>
      )}
    </GlassCard>
  );
};

interface MyReservationsProps {
  reservations: { id: string; deskLabel: string; date: string; time: string }[];
  onRelease: (id: string) => void | Promise<void>;
  releasingId?: string | null;
}

export const MyReservations = ({ reservations, onRelease, releasingId }: MyReservationsProps) => (
  <GlassCard variant="strong" className="p-6">
      <div className="flex items-center justify-between mb-4">
      <h3 className="font-display text-base font-semibold">Мои резервации</h3>
      <span className="glass rounded-full px-2.5 py-0.5 text-xs">{reservations.length}</span>
    </div>
    {reservations.length === 0 ? (
      <p className="text-sm text-muted-foreground text-center py-6">Резерваций пока нет</p>
    ) : (
      <div className="space-y-2">
        {reservations.map((r) => (
          <div key={r.id} className="glass glass-hover rounded-2xl p-3 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-primary/15 border border-primary/30 grid place-items-center font-display text-xs font-semibold text-primary">
                {r.deskLabel}
              </div>
              <div>
                <p className="text-sm font-medium">{r.deskLabel}</p>
                <p className="text-[11px] text-muted-foreground">{r.date} · {r.time}</p>
              </div>
            </div>
            <button
              onClick={() => onRelease(r.id)}
              disabled={releasingId === r.id}
              className="glass rounded-full p-1.5 hover:bg-destructive/20 hover:text-destructive transition-colors disabled:opacity-50"
              aria-label="Освободить"
            >
              {releasingId === r.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <X className="h-3.5 w-3.5" />}
            </button>
          </div>
        ))}
      </div>
    )}
  </GlassCard>
);