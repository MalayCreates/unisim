package store

import "time"

// These structs mirror the proto schema and are the canonical in-process types.
// Once protoc generation is wired up (scripts/gen-proto.sh), the generated
// types in internal/schema/ will be used directly; these will be removed.

type Side string

const (
	SideFriendly Side = "friendly"
	SideEnemy    Side = "enemy"
	SideNeutral  Side = "neutral"
)

type EntityType string

const (
	EntityTypeFixedWing          EntityType = "fixed_wing"
	EntityTypeRotaryWing         EntityType = "rotary_wing"
	EntityTypeGroundVehicle      EntityType = "ground_vehicle"
	EntityTypeDismountedInfantry EntityType = "dismounted_infantry"
	EntityTypeSurfaceVessel      EntityType = "surface_vessel"
	EntityTypeSubmarine          EntityType = "submarine"
	EntityTypeSatellite          EntityType = "satellite"
	EntityTypeUAV                EntityType = "uav"
	EntityTypeMissile            EntityType = "missile"
	EntityTypeRadarSensor        EntityType = "radar_sensor"
	EntityTypeBaseFOB            EntityType = "base_fob"
)

type Position struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	AltM float64 `json:"alt_m"`
}

type Entity struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       EntityType        `json:"type"`
	Side       Side              `json:"side"`
	Position   Position          `json:"position"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Waypoint struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	AltM      float64 `json:"alt_m"`
	SpeedMS   float64 `json:"speed_ms"`
	HoldTimeS uint32  `json:"hold_time_s"`
}

type ROE string

const (
	ROEWeaponsHold  ROE = "weapons_hold"
	ROEWeaponsTight ROE = "weapons_tight"
	ROEWeaponsFree  ROE = "weapons_free"
)

type MissionType string

const (
	MissionTypeCAP     MissionType = "cap"
	MissionTypeCAS     MissionType = "cas"
	MissionTypeStrike  MissionType = "strike"
	MissionTypePatrol  MissionType = "patrol"
	MissionTypeRecon   MissionType = "recon"
	MissionTypeEscort  MissionType = "escort"
	MissionTypeSAR     MissionType = "sar"
	MissionTypeTransit MissionType = "transit"
)

type MissionTimeline struct {
	StartOffsetS      uint32 `json:"start_offset_s"`
	ExpectedDurationS uint32 `json:"expected_duration_s"`
}

type EntityMission struct {
	EntityID    string          `json:"entity_id"`
	MissionType MissionType     `json:"mission_type"`
	Waypoints   []Waypoint      `json:"waypoints"`
	ROE         ROE             `json:"roe"`
	Objectives  []string        `json:"objectives"`
	Timeline    MissionTimeline `json:"timeline"`
}

type ScenarioSide struct {
	Affiliation Side   `json:"affiliation"`
	Label       string `json:"label"`
}

type TerrainReference struct {
	MinLat      float64 `json:"min_lat"`
	MaxLat      float64 `json:"max_lat"`
	MinLon      float64 `json:"min_lon"`
	MaxLon      float64 `json:"max_lon"`
	Description string  `json:"description"`
}

type Scenario struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Sides       []ScenarioSide    `json:"sides"`
	Entities    []Entity          `json:"entities"`
	Missions    []EntityMission   `json:"missions"`
	StartTime   time.Time         `json:"start_time"`
	DurationS   uint32            `json:"duration_s"`
	Terrain     *TerrainReference `json:"terrain,omitempty"`
	EngineHint  string            `json:"engine_hint"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

type Run struct {
	ID         string     `json:"id"`
	ScenarioID string     `json:"scenario_id"`
	EngineID   string     `json:"engine_id"`
	Status     RunStatus  `json:"status"`
	Error      string     `json:"error,omitempty"`
	Priority   int        `json:"priority"`
	WorkerID   string     `json:"worker_id,omitempty"`
	ClaimedAt  *time.Time `json:"claimed_at,omitempty"`
	// BatchID groups runs created together as Monte Carlo replications of the
	// same scenario (see api.batchHandler). Empty for a standalone run.
	BatchID   string    `json:"batch_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RunStatusQueued is the initial state for a run accepted into the dispatch
// queue but not yet claimed by a worker. It is an alias of pending so existing
// callers and the UI keep working; the dispatcher treats it as "ready to run".
const RunStatusQueued = RunStatusPending

type TrackPoint struct {
	TimestampMS int64   `json:"timestamp_ms"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	AltM        float64 `json:"alt_m"`
	HeadingDeg  float64 `json:"heading_deg"`
	SpeedMS     float64 `json:"speed_ms"`
	Status      string  `json:"status"`
}

type EntityTrack struct {
	EntityID string       `json:"entity_id"`
	Points   []TrackPoint `json:"points"`
}

type EventType string

const (
	EventTypeDetection       EventType = "detection"
	EventTypeEngagement      EventType = "engagement"
	EventTypeKill            EventType = "kill"
	EventTypeDamage          EventType = "damage"
	EventTypeLaunch          EventType = "launch"
	EventTypeWaypointReached EventType = "waypoint_reached"
	EventTypeMissionComplete EventType = "mission_complete"
)

type SimEvent struct {
	TimestampMS    int64     `json:"timestamp_ms"`
	Type           EventType `json:"type"`
	EntityID       string    `json:"entity_id"`
	TargetEntityID string    `json:"target_entity_id,omitempty"`
	Detail         string    `json:"detail,omitempty"`
}

type KillChain struct {
	AttackerEntityID string   `json:"attacker_entity_id"`
	TargetEntityID   string   `json:"target_entity_id"`
	EngagedAtMS      int64    `json:"engaged_at_ms"`
	KilledAtMS       int64    `json:"killed_at_ms"`
	WeaponIDs        []string `json:"weapon_ids,omitempty"`
}

type MOEMetric struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type SimResults struct {
	ID           string        `json:"id"`
	ScenarioID   string        `json:"scenario_id"`
	EngineID     string        `json:"engine_id"`
	RunID        string        `json:"run_id"`
	EntityTracks []EntityTrack `json:"entity_tracks"`
	Events       []SimEvent    `json:"events"`
	KillChains   []KillChain   `json:"kill_chains"`
	MOEMetrics   []MOEMetric   `json:"moe_metrics"`
	CreatedAt    time.Time     `json:"created_at"`
}
