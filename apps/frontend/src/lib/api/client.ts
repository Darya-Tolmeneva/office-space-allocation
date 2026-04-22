import type {
  AnalyticsSummary,
  AuthResponse,
  AutoReservationPreview,
  AutoReservationRequest,
  AvailabilitySlot,
  Desk,
  Floor,
  FloorWithDesks,
  Reservation,
  ReservationCreate,
  TokenPair,
} from "./types";

const TOKENS_KEY = "flowdesk.tokens";
const USER_KEY = "flowdesk.user";

export const API_BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") ||
  "/v1";

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export const tokenStorage = {
  get(): TokenPair | null {
    try {
      const raw = localStorage.getItem(TOKENS_KEY);
      return raw ? (JSON.parse(raw) as TokenPair) : null;
    } catch {
      return null;
    }
  },
  set(t: TokenPair) {
    localStorage.setItem(TOKENS_KEY, JSON.stringify(t));
  },
  clear() {
    localStorage.removeItem(TOKENS_KEY);
    localStorage.removeItem(USER_KEY);
  },
};

export const userStorage = {
  get<T>(): T | null {
    try {
      const raw = localStorage.getItem(USER_KEY);
      return raw ? (JSON.parse(raw) as T) : null;
    } catch {
      return null;
    }
  },
  set<T>(u: T) {
    localStorage.setItem(USER_KEY, JSON.stringify(u));
  },
};

let refreshPromise: Promise<TokenPair | null> | null = null;
let onUnauthorized: (() => void) | null = null;
export function setOnUnauthorized(fn: (() => void) | null) {
  onUnauthorized = fn;
}

async function refreshTokens(): Promise<TokenPair | null> {
  if (refreshPromise) return refreshPromise;
  const current = tokenStorage.get();
  if (!current?.refreshToken) return null;
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: current.refreshToken }),
      });
      if (!res.ok) return null;
      const data = (await res.json()) as AuthResponse;
      tokenStorage.set(data.tokens);
      userStorage.set(data.user);
      return data.tokens;
    } catch {
      return null;
    } finally {
      refreshPromise = null;
    }
  })();
  return refreshPromise;
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  auth?: boolean;
  query?: Record<string, string | number | boolean | undefined | null>;
  _retried?: boolean;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { body, auth = false, query, headers, _retried, ...rest } = opts;

  const base = API_BASE_URL.startsWith("http") ? API_BASE_URL : `${window.location.origin}${API_BASE_URL}`;
  const url = new URL(`${base}${path}`);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== "") url.searchParams.set(k, String(v));
    }
  }

  const finalHeaders: Record<string, string> = {
    Accept: "application/json",
    ...(headers as Record<string, string>),
  };
  if (body !== undefined) finalHeaders["Content-Type"] = "application/json";
  if (auth) {
    const t = tokenStorage.get();
    if (t?.accessToken) finalHeaders["Authorization"] = `Bearer ${t.accessToken}`;
  }

  const res = await fetch(url.toString(), {
    ...rest,
    headers: finalHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (res.status === 401 && auth && !_retried) {
    const refreshed = await refreshTokens();
    if (refreshed) {
      return request<T>(path, { ...opts, _retried: true });
    }
    tokenStorage.clear();
    onUnauthorized?.();
  }

  if (res.status === 204) return undefined as T;

  let data: unknown = null;
  const text = await res.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }

  if (!res.ok) {
    const errObj =
      (data as { error?: { code?: string; message?: string } })?.error ?? {};
    throw new ApiError(
      res.status,
      errObj.code ?? `http_${res.status}`,
      errObj.message ?? res.statusText ?? "Request failed",
    );
  }

  return data as T;
}

export const api = {
  // auth
  register: (b: { email: string; password: string; fullName: string }) =>
    request<AuthResponse>("/auth/register", { method: "POST", body: b }),
  login: (b: { email: string; password: string }) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: b }),

  // floors
  listFloors: () => request<Floor[]>("/floors"),
  getFloor: (floorId: string) =>
    request<FloorWithDesks>(`/floors/${encodeURIComponent(floorId)}`),

  // desks
  listDesks: (q?: { floorId?: string; zoneId?: string; features?: string }) =>
    request<Desk[]>("/desks", { query: q }),
  getDesk: (deskId: string) =>
    request<Desk>(`/desks/${encodeURIComponent(deskId)}`),
  getDeskAvailability: (deskId: string, from: string, to: string) =>
    request<AvailabilitySlot[]>(
      `/desks/${encodeURIComponent(deskId)}/availability`,
      { query: { from, to } },
    ),

  // reservations
  listReservations: (q?: {
    deskId?: string;
    floorId?: string;
    from?: string;
    to?: string;
    status?: "active" | "cancelled" | "completed" | "all";
  }) => request<Reservation[]>("/reservations", { auth: true, query: q }),
  createReservation: (b: ReservationCreate) =>
    request<Reservation>("/reservations", { method: "POST", auth: true, body: b }),
  autoReserve: (b: AutoReservationRequest) =>
    request<Reservation>("/reservations/auto", { method: "POST", auth: true, body: b }),
  autoReservePreview: (b: AutoReservationRequest) =>
    request<AutoReservationPreview>("/reservations/auto/preview", {
      method: "POST",
      auth: true,
      body: b,
    }),
  getReservation: (id: string) =>
    request<Reservation>(`/reservations/${encodeURIComponent(id)}`, { auth: true }),
  cancelReservation: (id: string) =>
    request<void>(`/reservations/${encodeURIComponent(id)}`, {
      method: "DELETE",
      auth: true,
    }),
  releaseReservation: (id: string, reason?: string) =>
    request<Reservation>(`/reservations/${encodeURIComponent(id)}/release`, {
      method: "POST",
      auth: true,
      body: reason ? { reason } : undefined,
    }),

  // analytics
  analyticsSummary: (q?: { floorId?: string; from?: string; to?: string }) =>
    request<AnalyticsSummary>("/analytics/summary", { query: q }),
};