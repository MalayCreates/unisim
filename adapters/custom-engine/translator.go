package main

import (
	"github.com/usip/backend/schema"
)

// simWaypoint is the engine's internal waypoint representation.
type simWaypoint struct {
	lat, lon, alt float64
	speedMS       float64
	holdTimeS     uint32
}

// simEntity is the engine's mutable per-entity state during a run.
type simEntity struct {
	id    string
	name  string
	side  schema.Side
	etype schema.EntityType

	// Live kinematic state
	lat, lon, alt float64
	headingDeg    float64
	speedMS       float64
	alive         bool

	// Mission
	waypoints  []simWaypoint
	roe        schema.ROE
	wpIndex    int
	holdUntilS float64 // sim-time second until which the entity loiters

	// Derived capabilities (meters)
	sensorRangeM float64
	weaponRangeM float64

	// Bookkeeping: enemies already detected (avoid duplicate detection events)
	detected map[string]bool
}

// capability holds the default sensor/weapon ranges and base lethality for a
// given entity type. These are deliberately simple — this is a reference engine.
type capability struct {
	sensorRangeM float64
	weaponRangeM float64
	basePk       float64 // base probability of kill when it engages
}

// capabilityTable maps entity type to default capabilities. Engine-specific
// overrides can later come from Entity.attributes.
var capabilityTable = map[schema.EntityType]capability{
	schema.EntityType_ENTITY_TYPE_FIXED_WING:     {sensorRangeM: 80000, weaponRangeM: 40000, basePk: 0.7},
	schema.EntityType_ENTITY_TYPE_ROTARY_WING:    {sensorRangeM: 30000, weaponRangeM: 8000, basePk: 0.6},
	schema.EntityType_ENTITY_TYPE_UAV:            {sensorRangeM: 60000, weaponRangeM: 15000, basePk: 0.5},
	schema.EntityType_ENTITY_TYPE_GROUND_VEHICLE: {sensorRangeM: 8000, weaponRangeM: 4000, basePk: 0.6},
	schema.EntityType_ENTITY_TYPE_SURFACE_VESSEL: {sensorRangeM: 100000, weaponRangeM: 50000, basePk: 0.65},
	schema.EntityType_ENTITY_TYPE_SUBMARINE:      {sensorRangeM: 40000, weaponRangeM: 20000, basePk: 0.7},
	schema.EntityType_ENTITY_TYPE_RADAR_SENSOR:   {sensorRangeM: 150000, weaponRangeM: 0, basePk: 0},
	schema.EntityType_ENTITY_TYPE_BASE_FOB:       {sensorRangeM: 50000, weaponRangeM: 0, basePk: 0},
	schema.EntityType_ENTITY_TYPE_MISSILE:        {sensorRangeM: 20000, weaponRangeM: 10000, basePk: 0.85},
}

// defaultCapability is used for any type not in the table.
var defaultCapability = capability{sensorRangeM: 10000, weaponRangeM: 5000, basePk: 0.5}

func capabilityFor(t schema.EntityType) capability {
	if c, ok := capabilityTable[t]; ok {
		return c
	}
	return defaultCapability
}

// translate converts a canonical ScenarioProto into the engine's internal
// entity list, attaching each entity's mission waypoints/ROE.
func translate(s *schema.ScenarioProto) []*simEntity {
	missionByEntity := make(map[string]*schema.EntityMission, len(s.Missions))
	for _, m := range s.Missions {
		missionByEntity[m.EntityId] = m
	}

	out := make([]*simEntity, 0, len(s.Entities))
	for _, e := range s.Entities {
		cap := capabilityFor(e.Type)
		se := &simEntity{
			id:           e.Id,
			name:         e.Name,
			side:         e.Side,
			etype:        e.Type,
			alive:        true,
			roe:          schema.ROE_ROE_WEAPONS_TIGHT,
			sensorRangeM: cap.sensorRangeM,
			weaponRangeM: cap.weaponRangeM,
			detected:     make(map[string]bool),
		}
		if e.Position != nil {
			se.lat = e.Position.Lat
			se.lon = e.Position.Lon
			se.alt = e.Position.AltM
		}
		if m := missionByEntity[e.Id]; m != nil {
			se.roe = m.Roe
			for _, w := range m.Waypoints {
				se.waypoints = append(se.waypoints, simWaypoint{
					lat:       w.Lat,
					lon:       w.Lon,
					alt:       w.AltM,
					speedMS:   w.SpeedMs,
					holdTimeS: w.HoldTimeS,
				})
			}
		}
		out = append(out, se)
	}
	return out
}
