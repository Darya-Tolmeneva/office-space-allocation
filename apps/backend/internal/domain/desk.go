package domain

import "time"

// DeskState describes the persisted lifecycle state of a desk.
type DeskState string

const (
	DeskStateActive   DeskState = "active"
	DeskStateDisabled DeskState = "disabled"
)

// DeskFeature describes a desk capability used in filtering and auto-pick.
type DeskFeature string

const (
	DeskFeatureMonitor     DeskFeature = "monitor"
	DeskFeatureDualMonitor DeskFeature = "dual_monitor"
	DeskFeatureWiFi        DeskFeature = "wifi"
	DeskFeatureEthernet    DeskFeature = "ethernet"
	DeskFeatureStanding    DeskFeature = "standing"
	DeskFeatureQuiet       DeskFeature = "quiet"
	DeskFeatureWindow      DeskFeature = "window"
	DeskFeatureNearKitchen DeskFeature = "near_kitchen"
	DeskFeatureAccessible  DeskFeature = "accessible"
)

// DeskPosition stores desk coordinates on the floor plan.
type DeskPosition struct {
	X float64
	Y float64
}

// Desk represents a reservable workplace.
type Desk struct {
	ID        string
	FloorID   string
	ZoneID    string
	Label     string
	State     DeskState
	Position  DeskPosition
	Features  []DeskFeature
	CreatedAt time.Time
}
