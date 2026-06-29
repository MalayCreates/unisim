package main

import (
	"testing"

	"github.com/usip/backend/schema"
)

// buildScenario returns a minimal blue-strikes-red scenario: a friendly
// fixed-wing with a waypoint on top of a stationary enemy vessel.
func buildScenario() *schema.ScenarioProto {
	return &schema.ScenarioProto{
		Id:        "test-scenario",
		Name:      "engine unit test",
		DurationS: 600,
		Entities: []*schema.Entity{
			{
				Id:       "blue1",
				Name:     "Viper 1",
				Type:     schema.EntityType_ENTITY_TYPE_FIXED_WING,
				Side:     schema.Side_SIDE_FRIENDLY,
				Position: &schema.Position{Lat: 0, Lon: 0.5, AltM: 8000},
			},
			{
				Id:       "red1",
				Name:     "Red Vessel",
				Type:     schema.EntityType_ENTITY_TYPE_SURFACE_VESSEL,
				Side:     schema.Side_SIDE_ENEMY,
				Position: &schema.Position{Lat: 0, Lon: 0, AltM: 0},
			},
		},
		Missions: []*schema.EntityMission{
			{
				EntityId:    "blue1",
				MissionType: schema.MissionType_MISSION_TYPE_STRIKE,
				Roe:         schema.ROE_ROE_WEAPONS_FREE,
				Waypoints: []*schema.Waypoint{
					{Lat: 0, Lon: 0, AltM: 8000, SpeedMs: 300},
				},
			},
		},
	}
}

func TestEngineProducesTracks(t *testing.T) {
	res := newEngine(buildScenario(), "run-1").run("test-scenario", engineID, "run-1")

	if got := len(res.EntityTracks); got != 2 {
		t.Fatalf("expected 2 entity tracks, got %d", got)
	}
	for _, tr := range res.EntityTracks {
		if len(tr.Points) == 0 {
			t.Errorf("entity %s has no track points", tr.EntityId)
		}
	}
}

func TestBlueMovesTowardTarget(t *testing.T) {
	res := newEngine(buildScenario(), "run-1").run("test-scenario", engineID, "run-1")

	var blue *schema.EntityTrack
	for _, tr := range res.EntityTracks {
		if tr.EntityId == "blue1" {
			blue = tr
		}
	}
	if blue == nil {
		t.Fatal("no track for blue1")
	}
	first, last := blue.Points[0], blue.Points[len(blue.Points)-1]
	if last.Lon >= first.Lon {
		t.Errorf("blue1 should move west (decreasing lon): start=%.4f end=%.4f", first.Lon, last.Lon)
	}
}

func TestEngagementProducesEventsAndIsDeterministic(t *testing.T) {
	// Same run ID -> same RNG seed -> identical kill count across runs.
	a := newEngine(buildScenario(), "run-det").run("test-scenario", engineID, "run-det")
	b := newEngine(buildScenario(), "run-det").run("test-scenario", engineID, "run-det")

	if len(a.Events) == 0 {
		t.Error("expected at least one event (detection/engagement)")
	}
	if len(a.KillChains) != len(b.KillChains) {
		t.Errorf("run not deterministic: kill counts %d vs %d", len(a.KillChains), len(b.KillChains))
	}

	// A detection should occur since the two entities start within sensor range.
	var sawDetection bool
	for _, e := range a.Events {
		if e.Type == schema.EventType_EVENT_TYPE_DETECTION {
			sawDetection = true
		}
	}
	if !sawDetection {
		t.Error("expected a detection event")
	}
}
