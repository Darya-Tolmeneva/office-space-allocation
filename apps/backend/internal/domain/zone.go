package domain

import "time"

// ZoneType describes the type of a floor zone.
type ZoneType string

const (
	ZoneTypeOpenSpace  ZoneType = "open_space"
	ZoneTypeMeeting    ZoneType = "meeting_room"
	ZoneTypePhoneBooth ZoneType = "phone_booth"
	ZoneTypeQuietZone  ZoneType = "quiet_zone"
)

// Zone groups desks within a floor.
type Zone struct {
	ID        string
	FloorID   string
	Name      string
	Type      ZoneType
	CreatedAt time.Time
}
