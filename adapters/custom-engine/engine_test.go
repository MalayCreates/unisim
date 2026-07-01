package main

import (
	"math"
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

// buildScenarioWithAttrs is buildScenario but with an attributes map attached
// to the entity at entityIdx, so tests can exercise per-entity capability
// overrides in isolation.
func buildScenarioWithAttrs(entityIdx int, attrs map[string]string) *schema.ScenarioProto {
	s := buildScenario()
	s.Entities[entityIdx].Attributes = attrs
	return s
}

func TestHealthOverrideSurvivesMultipleHits(t *testing.T) {
	// Give red1 (index 1) a guaranteed hit (base_pk=1) and blue1 (index 0)
	// 3 HP, so the outcome is deterministic rather than depending on pk
	// rolls: red gets its first engagement window (its 50km weapon range vs
	// blue's 40km) well before blue can fire back, so it lands exactly the
	// 3 unanswered guaranteed hits needed to produce 2 DAMAGE events then a
	// KILL, regardless of RNG seed.
	s := buildScenario()
	s.Entities[0].Attributes = map[string]string{"health_hp": "3"}
	s.Entities[1].Attributes = map[string]string{"base_pk": "1"}

	res := newEngine(s, "run-health").run("test-scenario", engineID, "run-health")

	var damageEvents, killEvents int
	for _, e := range res.Events {
		if e.TargetEntityId != "blue1" {
			continue
		}
		switch e.Type {
		case schema.EventType_EVENT_TYPE_DAMAGE:
			damageEvents++
		case schema.EventType_EVENT_TYPE_KILL:
			killEvents++
		}
	}
	if damageEvents != 2 {
		t.Errorf("expected exactly 2 DAMAGE events against the 3-HP entity before it dies, got %d", damageEvents)
	}
	if killEvents != 1 {
		t.Errorf("expected exactly one KILL event for the 3-HP entity, got %d", killEvents)
	}
}

func TestAmmoExhaustionStopsEngagement(t *testing.T) {
	// blue1 (index 0) is the primary attacker in this scenario; giving it
	// zero ammo should mean it never fires, regardless of what red1 does.
	res := newEngine(buildScenarioWithAttrs(0, map[string]string{"ammo": "0"}), "run-ammo").
		run("test-scenario", engineID, "run-ammo")

	for _, e := range res.Events {
		if e.EntityId == "blue1" && (e.Type == schema.EventType_EVENT_TYPE_ENGAGEMENT || e.Type == schema.EventType_EVENT_TYPE_KILL) {
			t.Errorf("expected blue1 to never fire with ammo=0, got event %v", e.Type)
		}
	}
	for _, kc := range res.KillChains {
		if kc.AttackerEntityId == "blue1" {
			t.Error("expected no kill chains with blue1 as attacker when ammo=0")
		}
	}
}

func TestCapabilityAttributeOverrides(t *testing.T) {
	s := buildScenarioWithAttrs(1, map[string]string{
		"sensor_range_m": "1",
		"weapon_range_m": "1",
	})
	ents := translate(s)
	var red *simEntity
	for _, e := range ents {
		if e.id == "red1" {
			red = e
		}
	}
	if red == nil {
		t.Fatal("red1 not found after translate")
	}
	if red.sensorRangeM != 1 || red.weaponRangeM != 1 {
		t.Errorf("expected overridden ranges of 1m, got sensor=%.0f weapon=%.0f", red.sensorRangeM, red.weaponRangeM)
	}
}

func TestComputeMOEsCanonicalMetrics(t *testing.T) {
	res := newEngine(buildScenario(), "run-moe").run("test-scenario", engineID, "run-moe")

	moes := make(map[string]float64, len(res.MoeMetrics))
	for _, m := range res.MoeMetrics {
		moes[m.Key] = m.Value
	}
	for _, key := range []string{
		"blue_losses", "red_losses", "blue_kills", "red_kills", "total_kills",
		"detections_total", "rounds_expended", "avg_health_pct",
	} {
		if _, ok := moes[key]; !ok {
			t.Errorf("missing MOE key %q", key)
		}
	}

	var wantDetections, wantEngagements float64
	for _, e := range res.Events {
		switch e.Type {
		case schema.EventType_EVENT_TYPE_DETECTION:
			wantDetections++
		case schema.EventType_EVENT_TYPE_ENGAGEMENT:
			wantEngagements++
		}
	}
	if moes["detections_total"] != wantDetections {
		t.Errorf("detections_total = %v, want %v (count of DETECTION events)", moes["detections_total"], wantDetections)
	}
	if moes["rounds_expended"] != wantEngagements {
		t.Errorf("rounds_expended = %v, want %v (count of ENGAGEMENT events)", moes["rounds_expended"], wantEngagements)
	}
	if moes["avg_health_pct"] < 0 || moes["avg_health_pct"] > 100 {
		t.Errorf("avg_health_pct = %v, want a value in [0, 100]", moes["avg_health_pct"])
	}
}

func TestLerp(t *testing.T) {
	cases := []struct {
		a, b, frac, want float64
	}{
		{1, 0, 0, 1},
		{1, 0, 1, 0},
		{1, 0, 0.5, 0.5},
		{1, 0.2, -1, 1},  // clamps below 0
		{1, 0.2, 2, 0.2}, // clamps above 1
	}
	for _, c := range cases {
		if got := lerp(c.a, c.b, c.frac); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("lerp(%v, %v, %v) = %v, want %v", c.a, c.b, c.frac, got, c.want)
		}
	}
}
