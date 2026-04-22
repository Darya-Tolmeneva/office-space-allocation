import { cn } from "@/lib/utils";
import { Coffee } from "lucide-react";
import type { DeskFeature, Zone } from "@/lib/api/types";

export type DeskStatus = "available" | "reserved" | "occupied" | "mine" | "disabled";

export interface Desk {
  id: string;
  label: string;
  x: number;
  y: number;
  status: DeskStatus;
  features: DeskFeature[];
}

interface FloorPlanProps {
  desks: Desk[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  floorName?: string;
  zones?: Zone[];
}

const statusStyles: Record<DeskStatus, string> = {
  available: "bg-white/80 border-border text-muted-foreground hover:border-primary hover:text-primary hover:shadow-md",
  reserved: "bg-warning/15 border-warning/40 text-warning/90",
  occupied: "bg-muted border-border/60 text-muted-foreground/50 cursor-not-allowed",
  mine: "bg-primary text-primary-foreground border-primary shadow-lg",
  disabled: "bg-muted/50 border-border/40 text-muted-foreground/40 cursor-not-allowed",
};

const zoneTypeLabel: Record<Zone["type"], string> = {
  open_space: "Open Space",
  meeting_room: "Переговорка",
  phone_booth: "Phone Booth",
  quiet_zone: "Тихая зона",
};

export const FloorPlan = ({ desks, selectedId, onSelect, floorName, zones = [] }: FloorPlanProps) => {
  const meetingRooms = zones.filter((z) => z.type === "meeting_room");
  const primaryZone = zones.find((z) => z.type !== "meeting_room");
  return (
    <div className="relative w-full aspect-[16/10] rounded-3xl overflow-hidden glass-strong p-6">
      {/* Soft ambient blobs */}
      <div className="absolute -top-32 -right-32 h-80 w-80 rounded-full bg-primary/15 blur-3xl animate-blob" />
      <div className="absolute -bottom-32 -left-32 h-80 w-80 rounded-full bg-accent/15 blur-3xl animate-blob" style={{ animationDelay: "6s" }} />

      {/* Grid background */}
      <div
        className="absolute inset-0 opacity-[0.07]"
        style={{
          backgroundImage:
            "linear-gradient(hsl(var(--foreground)) 1px, transparent 1px), linear-gradient(90deg, hsl(var(--foreground)) 1px, transparent 1px)",
          backgroundSize: "40px 40px",
        }}
      />

      {/* Zone labels */}
      <div className="absolute top-6 left-6 glass rounded-full px-3 py-1 text-[10px] uppercase tracking-widest text-muted-foreground">
        {primaryZone ? `${zoneTypeLabel[primaryZone.type]} · ${primaryZone.name}` : floorName ?? "Этаж"}
      </div>
      <div className="absolute top-6 right-6 glass rounded-full px-3 py-1.5 flex items-center gap-2 text-xs">
        <Coffee className="h-3 w-3 text-muted-foreground" />
        <span className="text-muted-foreground">{floorName ?? "Этаж"}</span>
      </div>

      {/* Meeting rooms — chips in the bottom-left, since API has no geometry */}
      {meetingRooms.length > 0 && (
        <div className="absolute bottom-16 left-6 flex flex-wrap gap-2 max-w-[60%]">
          {meetingRooms.map((room) => (
            <div key={room.id} className="glass rounded-full px-3 py-1.5 flex items-center gap-2 text-xs">
              <span className="h-1.5 w-1.5 rounded-full bg-accent" />
              <span className="text-[10px] uppercase tracking-widest text-muted-foreground">
                {zoneTypeLabel[room.type]}
              </span>
              <span className="font-display text-foreground">{room.name}</span>
            </div>
          ))}
        </div>
      )}

      {/* Desks */}
      {desks.map((desk) => {
        const isSelected = selectedId === desk.id;
        const disabled = desk.status === "occupied" || desk.status === "disabled";
        return (
          <button
            key={desk.id}
            disabled={disabled}
            onClick={() => onSelect(desk.id)}
            style={{ left: `${desk.x}%`, top: `${desk.y}%` }}
            className={cn(
              "absolute -translate-x-1/2 -translate-y-1/2 h-12 w-12 rounded-xl border backdrop-blur-md transition-all duration-200 grid place-items-center",
              statusStyles[desk.status],
              isSelected && "scale-110 ring-2 ring-primary/60 ring-offset-2 ring-offset-background z-10"
            )}
          >
            <span className="font-display text-xs font-bold">{desk.label}</span>
            {isSelected && (
              <span className="absolute -top-9 left-1/2 -translate-x-1/2 glass-strong rounded-lg px-2 py-1 text-[10px] whitespace-nowrap text-foreground">
                {desk.label}
              </span>
            )}
          </button>
        );
      })}

      {/* Legend */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 glass rounded-full px-4 py-2 flex items-center gap-4 text-xs">
        {[
          { c: "bg-white border border-border", l: "Свободно" },
          { c: "bg-warning/40 border border-warning/50", l: "Резерв" },
          { c: "bg-muted border border-border", l: "Занято" },
          { c: "bg-primary border border-primary", l: "Моё" },
        ].map((s) => (
          <div key={s.l} className="flex items-center gap-1.5">
            <span className={cn("h-2 w-2 rounded-full", s.c)} />
            <span className="text-muted-foreground">{s.l}</span>
          </div>
        ))}
      </div>
    </div>
  );
};