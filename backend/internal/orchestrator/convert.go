package orchestrator

import (
	"github.com/usip/backend/internal/store"
	"github.com/usip/backend/schema"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// scenarioToProto converts a stored Scenario into the canonical ScenarioProto
// delivered to adapters via Initialize().
func scenarioToProto(s *store.Scenario) *schema.ScenarioProto {
	p := &schema.ScenarioProto{
		Id:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		DurationS:   s.DurationS,
		EngineHint:  s.EngineHint,
	}
	if !s.StartTime.IsZero() {
		p.StartTime = timestamppb.New(s.StartTime)
	}
	for _, sd := range s.Sides {
		p.Sides = append(p.Sides, &schema.ScenarioSide{
			Affiliation: sideToProto(sd.Affiliation),
			Label:       sd.Label,
		})
	}
	for _, e := range s.Entities {
		p.Entities = append(p.Entities, &schema.Entity{
			Id:   e.ID,
			Name: e.Name,
			Type: entityTypeToProto(e.Type),
			Side: sideToProto(e.Side),
			Position: &schema.Position{
				Lat:  e.Position.Lat,
				Lon:  e.Position.Lon,
				AltM: e.Position.AltM,
			},
			Attributes: e.Attributes,
		})
	}
	for _, m := range s.Missions {
		em := &schema.EntityMission{
			EntityId:    m.EntityID,
			MissionType: missionTypeToProto(m.MissionType),
			Roe:         roeToProto(m.ROE),
			Objectives:  m.Objectives,
			Timeline: &schema.MissionTimeline{
				StartOffsetS:      m.Timeline.StartOffsetS,
				ExpectedDurationS: m.Timeline.ExpectedDurationS,
			},
		}
		for _, w := range m.Waypoints {
			em.Waypoints = append(em.Waypoints, &schema.Waypoint{
				Lat:       w.Lat,
				Lon:       w.Lon,
				AltM:      w.AltM,
				SpeedMs:   w.SpeedMS,
				HoldTimeS: w.HoldTimeS,
			})
		}
		p.Missions = append(p.Missions, em)
	}
	return p
}

func sideToProto(s store.Side) schema.Side {
	switch s {
	case store.SideFriendly:
		return schema.Side_SIDE_FRIENDLY
	case store.SideEnemy:
		return schema.Side_SIDE_ENEMY
	case store.SideNeutral:
		return schema.Side_SIDE_NEUTRAL
	default:
		return schema.Side_SIDE_UNSPECIFIED
	}
}

func entityTypeToProto(t store.EntityType) schema.EntityType {
	switch t {
	case store.EntityTypeFixedWing:
		return schema.EntityType_ENTITY_TYPE_FIXED_WING
	case store.EntityTypeRotaryWing:
		return schema.EntityType_ENTITY_TYPE_ROTARY_WING
	case store.EntityTypeGroundVehicle:
		return schema.EntityType_ENTITY_TYPE_GROUND_VEHICLE
	case store.EntityTypeDismountedInfantry:
		return schema.EntityType_ENTITY_TYPE_DISMOUNTED_INFANTRY
	case store.EntityTypeSurfaceVessel:
		return schema.EntityType_ENTITY_TYPE_SURFACE_VESSEL
	case store.EntityTypeSubmarine:
		return schema.EntityType_ENTITY_TYPE_SUBMARINE
	case store.EntityTypeSatellite:
		return schema.EntityType_ENTITY_TYPE_SATELLITE
	case store.EntityTypeUAV:
		return schema.EntityType_ENTITY_TYPE_UAV
	case store.EntityTypeMissile:
		return schema.EntityType_ENTITY_TYPE_MISSILE
	case store.EntityTypeRadarSensor:
		return schema.EntityType_ENTITY_TYPE_RADAR_SENSOR
	case store.EntityTypeBaseFOB:
		return schema.EntityType_ENTITY_TYPE_BASE_FOB
	default:
		return schema.EntityType_ENTITY_TYPE_UNSPECIFIED
	}
}

func missionTypeToProto(t store.MissionType) schema.MissionType {
	switch t {
	case store.MissionTypeCAP:
		return schema.MissionType_MISSION_TYPE_CAP
	case store.MissionTypeCAS:
		return schema.MissionType_MISSION_TYPE_CAS
	case store.MissionTypeStrike:
		return schema.MissionType_MISSION_TYPE_STRIKE
	case store.MissionTypePatrol:
		return schema.MissionType_MISSION_TYPE_PATROL
	case store.MissionTypeRecon:
		return schema.MissionType_MISSION_TYPE_RECON
	case store.MissionTypeEscort:
		return schema.MissionType_MISSION_TYPE_ESCORT
	case store.MissionTypeSAR:
		return schema.MissionType_MISSION_TYPE_SAR
	case store.MissionTypeTransit:
		return schema.MissionType_MISSION_TYPE_TRANSIT
	default:
		return schema.MissionType_MISSION_TYPE_UNSPECIFIED
	}
}

func roeToProto(r store.ROE) schema.ROE {
	switch r {
	case store.ROEWeaponsHold:
		return schema.ROE_ROE_WEAPONS_HOLD
	case store.ROEWeaponsTight:
		return schema.ROE_ROE_WEAPONS_TIGHT
	case store.ROEWeaponsFree:
		return schema.ROE_ROE_WEAPONS_FREE
	default:
		return schema.ROE_ROE_WEAPONS_TIGHT
	}
}
