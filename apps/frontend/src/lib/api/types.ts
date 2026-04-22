// Auto-generated from FlowDesk OpenAPI v1.2.0

export type DeskState = "active" | "disabled";
export type DeskRuntimeStatus = "available" | "reserved" | "occupied" | "disabled";
export type DeskFeature =
  | "monitor"
  | "dual_monitor"
  | "wifi"
  | "ethernet"
  | "standing"
  | "quiet"
  | "window"
  | "near_kitchen"
  | "accessible";

export interface AuthUser {
  id: string;
  email: string;
  fullName: string;
  role: "member" | "admin";
  status: "active" | "disabled";
  createdAt: string;
  updatedAt: string;
}

export interface TokenPair {
  accessToken: string;
  accessTokenExpiresAt: string;
  refreshToken: string;
  refreshTokenExpiresAt: string;
}

export interface AuthResponse {
  user: AuthUser;
  tokens: TokenPair;
}

export interface Zone {
  id: string;
  floorId: string;
  name: string;
  type: "open_space" | "meeting_room" | "phone_booth" | "quiet_zone";
  createdAt: string;
}

export interface DeskPosition {
  x: number;
  y: number;
}

export interface Desk {
  id: string;
  floorId: string;
  zoneId?: string | null;
  label: string;
  status: DeskState;
  position: DeskPosition;
  features: DeskFeature[];
  createdAt: string;
}

export interface Floor {
  id: string;
  name: string;
  timezone: string;
  floorPlanUrl?: string | null;
  createdAt: string;
  zones?: Zone[];
}

export interface FloorWithDesks extends Floor {
  desks: Desk[];
}

export interface AvailabilitySlot {
  from: string;
  to: string;
  status: DeskRuntimeStatus;
  reservationId?: string | null;
}

export interface Reservation {
  id: string;
  deskId: string;
  deskLabel: string;
  floorId: string;
  from: string;
  to: string;
  status: "active" | "completed" | "cancelled";
  source: "manual" | "auto";
  holderName?: string | null;
  note?: string | null;
  createdAt: string;
  cancelledAt?: string | null;
}

export interface ReservationCreate {
  deskId: string;
  from: string;
  to: string;
  note?: string | null;
}

export interface AutoReservationRequest {
  from: string;
  to: string;
  floorId?: string | null;
  zoneId?: string | null;
  requiredFeatures?: DeskFeature[];
}

export interface AutoReservationPreview {
  desk: Desk;
  score: number;
  reasons: string[];
}

export interface AnalyticsSummary {
  averageOccupancy?: number;
  peakDay?: string;
  peakOccupancy?: number;
  autoPickRatio?: number;
  totalReservations?: number;
  earlyReleases?: number;
  topZone?: { zoneId: string; name: string };
}

export interface ApiError {
  error: { code: string; message: string };
}