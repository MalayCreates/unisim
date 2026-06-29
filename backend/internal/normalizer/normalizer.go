// Package normalizer maps engine output (the canonical SimResultsProto returned
// by an adapter) into the store's persisted SimResults representation.
//
// Every adapter already returns SimResultsProto, so normalization here is a
// straight proto -> storage-model translation. If a future adapter returns
// engine-native shapes, the per-engine mapping belongs in that adapter, not here.
package normalizer

import (
	"github.com/google/uuid"
	"github.com/usip/backend/internal/store"
	"github.com/usip/backend/schema"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FromProto converts a SimResultsProto into a store.SimResults.
func FromProto(p *schema.SimResultsProto) *store.SimResults {
	out := &store.SimResults{
		ID:         uuid.NewString(),
		ScenarioID: p.ScenarioId,
		EngineID:   p.EngineId,
		RunID:      p.RunId,
	}

	for _, t := range p.EntityTracks {
		track := store.EntityTrack{EntityID: t.EntityId}
		for _, pt := range t.Points {
			track.Points = append(track.Points, store.TrackPoint{
				TimestampMS: tsMS(pt.Ts),
				Lat:         pt.Lat,
				Lon:         pt.Lon,
				AltM:        pt.AltM,
				HeadingDeg:  pt.HeadingDeg,
				SpeedMS:     pt.SpeedMs,
				Status:      pt.Status,
			})
		}
		out.EntityTracks = append(out.EntityTracks, track)
	}

	for _, e := range p.Events {
		out.Events = append(out.Events, store.SimEvent{
			TimestampMS:    tsMS(e.Ts),
			Type:           eventTypeFromProto(e.Type),
			EntityID:       e.EntityId,
			TargetEntityID: e.TargetEntityId,
			Detail:         e.Detail,
		})
	}

	for _, kc := range p.KillChains {
		out.KillChains = append(out.KillChains, store.KillChain{
			AttackerEntityID: kc.AttackerEntityId,
			TargetEntityID:   kc.TargetEntityId,
			EngagedAtMS:      tsMS(kc.EngagedAt),
			KilledAtMS:       tsMS(kc.KilledAt),
			WeaponIDs:        kc.WeaponIds,
		})
	}

	for _, m := range p.MoeMetrics {
		out.MOEMetrics = append(out.MOEMetrics, store.MOEMetric{
			Key:   m.Key,
			Value: m.Value,
			Unit:  m.Unit,
		})
	}

	return out
}

func tsMS(ts *timestamppb.Timestamp) int64 {
	if ts == nil {
		return 0
	}
	return ts.AsTime().UnixMilli()
}

func eventTypeFromProto(t schema.EventType) store.EventType {
	switch t {
	case schema.EventType_EVENT_TYPE_DETECTION:
		return store.EventTypeDetection
	case schema.EventType_EVENT_TYPE_ENGAGEMENT:
		return store.EventTypeEngagement
	case schema.EventType_EVENT_TYPE_KILL:
		return store.EventTypeKill
	case schema.EventType_EVENT_TYPE_DAMAGE:
		return store.EventTypeDamage
	case schema.EventType_EVENT_TYPE_LAUNCH:
		return store.EventTypeLaunch
	case schema.EventType_EVENT_TYPE_WAYPOINT_REACHED:
		return store.EventTypeWaypointReached
	case schema.EventType_EVENT_TYPE_MISSION_COMPLETE:
		return store.EventTypeMissionComplete
	default:
		return ""
	}
}
