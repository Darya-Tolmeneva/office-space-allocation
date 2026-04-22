import { useCallback, useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Header, type View } from "@/components/Header";
import { FloorPlan, type Desk as UIDesk, type DeskStatus as UIDeskStatus } from "@/components/FloorPlan";
import { BookingPanel, MyReservations } from "@/components/BookingPanel";
import { StatsBar } from "@/components/StatsBar";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api/client";
import type { AnalyticsSummary, FloorWithDesks, Reservation } from "@/lib/api/types";

interface UIReservation {
  id: string;
  deskId: string;
  deskLabel: string;
  date: string;
  time: string;
}

const fmtTime = (iso: string) => new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
const fmtDate = (iso: string) => {
  const d = new Date(iso);
  const today = new Date();
  const sameDay = d.toDateString() === today.toDateString();
  return sameDay ? "Сегодня" : d.toLocaleDateString();
};

/** Build today's 09:00–18:00 ISO range in the user's local timezone. */
const buildTodayRange = () => {
  const from = new Date();
  from.setHours(9, 0, 0, 0);
  const to = new Date();
  to.setHours(18, 0, 0, 0);
  return { from: from.toISOString(), to: to.toISOString() };
};

const reservationToUI = (r: Reservation): UIReservation => ({
  id: r.id,
  deskId: r.deskId,
  deskLabel: r.deskLabel,
  date: fmtDate(r.from),
  time: `${fmtTime(r.from)}–${fmtTime(r.to)}`,
});

const Index = () => {
  const [floor, setFloor] = useState<FloorWithDesks | null>(null);
  const [reservations, setReservations] = useState<Reservation[]>([]);
  const [analytics, setAnalytics] = useState<AnalyticsSummary | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [view, setView] = useState<View>("map");
  const [loading, setLoading] = useState(true);
  const [reserving, setReserving] = useState(false);
  const [autoPicking, setAutoPicking] = useState(false);
  const [releasingId, setReleasingId] = useState<string | null>(null);

  // Initial load: pick first floor, fetch its desks + my active reservations + analytics.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        setLoading(true);
        const floors = await api.listFloors();
        if (!floors.length) {
          if (!cancelled) toast.error("На сервере нет ни одного этажа");
          return;
        }
        const [floorData, myRes, summary] = await Promise.all([
          api.getFloor(floors[0].id),
          api.listReservations({ status: "active" }).catch(() => [] as Reservation[]),
          api.analyticsSummary({ floorId: floors[0].id }).catch(() => null),
        ]);
        if (cancelled) return;
        setFloor(floorData);
        setReservations(myRes);
        setAnalytics(summary);
      } catch (err) {
        if (!cancelled) {
          const msg = err instanceof ApiError ? err.message : "Не удалось загрузить данные";
          toast.error(msg);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Map API desks → UI desks, layering in runtime status from my reservations.
  const uiDesks: UIDesk[] = useMemo(() => {
    if (!floor) return [];
    const myDeskIds = new Set(reservations.filter((r) => r.status === "active").map((r) => r.deskId));
    return floor.desks.map((d) => {
      let status: UIDeskStatus;
      if (d.status === "disabled") status = "disabled";
      else if (myDeskIds.has(d.id)) status = "mine";
      else status = "available";
      return {
        id: d.id,
        label: d.label,
        x: d.position.x,
        y: d.position.y,
        status,
        features: d.features,
      };
    });
  }, [floor, reservations]);

  const selected = useMemo(() => uiDesks.find((d) => d.id === selectedId) ?? null, [uiDesks, selectedId]);

  const totals = useMemo(() => {
    const total = uiDesks.length;
    const reserved = reservations.filter((r) => r.status === "active").length;
    const free = uiDesks.filter((d) => d.status === "available").length;
    return { total, reserved, free };
  }, [uiDesks, reservations]);

  const reserve = useCallback(
    async (id: string) => {
      const desk = uiDesks.find((d) => d.id === id);
      if (!desk) return;
      const { from, to } = buildTodayRange();
      setReserving(true);
      try {
        const res = await api.createReservation({ deskId: id, from, to });
        setReservations((prev) => [res, ...prev]);
        setSelectedId(null);
        toast.success(`Место ${desk.label} забронировано`);
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : "Не удалось забронировать";
        toast.error(msg);
      } finally {
        setReserving(false);
      }
    },
    [uiDesks],
  );

  const release = useCallback(
    async (resId: string) => {
      const r = reservations.find((x) => x.id === resId);
      if (!r) return;
      setReleasingId(resId);
      try {
        await api.cancelReservation(resId);
        setReservations((prev) => prev.filter((x) => x.id !== resId));
        toast(`Место ${r.deskLabel} освобождено`);
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : "Не удалось освободить место";
        toast.error(msg);
      } finally {
        setReleasingId(null);
      }
    },
    [reservations],
  );

  const autoPick = useCallback(async () => {
    if (!floor) return;
    const { from, to } = buildTodayRange();
    setAutoPicking(true);
    try {
      const res = await api.autoReserve({ from, to, floorId: floor.id });
      setReservations((prev) => [res, ...prev]);
      setSelectedId(res.deskId);
      toast.success(`Сервер подобрал место ${res.deskLabel}`);
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : "Свободных мест нет";
      toast.error(msg);
    } finally {
      setAutoPicking(false);
    }
  }, [floor]);

  const uiReservations = useMemo(() => reservations.map(reservationToUI), [reservations]);

  return (
    <div className="min-h-screen pb-12">
      <Header view={view} onViewChange={setView} />
      <main className="px-4 lg:px-8 mt-8 max-w-[1600px] mx-auto space-y-6">
        {/* Hero */}
        <section className="flex flex-col lg:flex-row lg:items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-muted-foreground mb-2">
              {floor ? `${floor.name} · Live` : "Гибкие рабочие места · Live"}
            </p>
            <h1 className="font-display text-4xl lg:text-5xl font-semibold leading-[1.1] tracking-tight">
              Найди своё <span className="text-primary">идеальное</span><br />место за секунды
            </h1>
          </div>
          <p className="text-sm text-muted-foreground max-w-sm">
            Бронируйте стол вручную на карте офиса или доверьте выбор серверу — он подберёт первое
            подходящее место по вашим фильтрам.
          </p>
        </section>

        <StatsBar free={totals.free} total={totals.total} reserved={totals.reserved} occupancy={analytics?.averageOccupancy} />

        <AnimatePresence mode="wait">
          {view === "map" && (
            <motion.section
              key="map"
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.3 }}
              className="grid lg:grid-cols-[1fr_400px] gap-6"
            >
              {loading ? (
                <div className="glass-strong rounded-3xl aspect-[16/10] grid place-items-center text-sm text-muted-foreground">
                  Загружаем план этажа…
                </div>
              ) : (
                <FloorPlan
                  desks={uiDesks}
                  selectedId={selectedId}
                  onSelect={setSelectedId}
                  floorName={floor?.name}
                  zones={floor?.zones}
                />
              )}
              <div className="space-y-6">
                <BookingPanel
                  desk={selected}
                  onReserve={reserve}
                  onAutoPick={autoPick}
                  reserving={reserving}
                  autoPicking={autoPicking}
                />
                <MyReservations reservations={uiReservations} onRelease={release} releasingId={releasingId} />
              </div>
            </motion.section>
          )}
          {view === "reservations" && (
            <motion.section
              key="reservations"
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.3 }}
              className="max-w-2xl mx-auto w-full"
            >
              <MyReservations reservations={uiReservations} onRelease={release} releasingId={releasingId} />
            </motion.section>
          )}
          {view === "analytics" && (
            <motion.section
              key="analytics"
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.3 }}
              className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4"
            >
              {[
                { l: "Средняя загрузка", v: analytics?.averageOccupancy !== undefined ? `${Math.round(analytics.averageOccupancy * 100)}%` : "—", d: "за период" },
                { l: "Пиковый день", v: analytics?.peakDay ?? "—", d: analytics?.peakOccupancy !== undefined ? `${Math.round(analytics.peakOccupancy * 100)}% мест занято` : "—" },
                { l: "Любимая зона", v: analytics?.topZone?.name ?? "—", d: "по броням команды" },
                { l: "Авто-подбор", v: analytics?.autoPickRatio !== undefined ? `${Math.round(analytics.autoPickRatio * 100)}%` : "—", d: "доля автоматических броней" },
                { l: "Всего бронирований", v: analytics?.totalReservations ?? "—", d: "за период" },
                { l: "Освобождений", v: analytics?.earlyReleases ?? "—", d: "досрочных за период" },
              ].map((c) => (
                <div key={c.l} className="glass-strong rounded-3xl p-5">
                  <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{c.l}</p>
                  <p className="font-display text-3xl font-semibold mt-2">{c.v}</p>
                  <p className="text-xs text-muted-foreground mt-1">{c.d}</p>
                </div>
              ))}
            </motion.section>
          )}
        </AnimatePresence>

        <footer className="pt-8 text-center text-xs text-muted-foreground">
          FlowDesk · Manufactory Aero edition
        </footer>
      </main>
    </div>
  );
};

export default Index;