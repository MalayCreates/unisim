package main

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/usip/backend/schema"
)

// genScenario builds an n-entity scenario spread over a lat/lon box, with
// alternating sides and one waypoint each, for benchmarking and for the
// grid-vs-brute equality test. Deterministic for a given seed.
func genScenario(n int, seed int64) *schema.ScenarioProto {
	r := rand.New(rand.NewSource(seed))
	s := &schema.ScenarioProto{Id: "bench", Name: "bench", DurationS: 120}
	types := []schema.EntityType{
		schema.EntityType_ENTITY_TYPE_FIXED_WING,
		schema.EntityType_ENTITY_TYPE_SURFACE_VESSEL,
		schema.EntityType_ENTITY_TYPE_GROUND_VEHICLE,
	}
	for i := 0; i < n; i++ {
		side := schema.Side_SIDE_FRIENDLY
		if i%2 == 1 {
			side = schema.Side_SIDE_ENEMY
		}
		lat := 20.0 + r.Float64()*10 // 20..30
		lon := 50.0 + r.Float64()*10 // 50..60
		id := fmt.Sprintf("e%d", i)
		s.Entities = append(s.Entities, &schema.Entity{
			Id:       id,
			Name:     id,
			Type:     types[i%len(types)],
			Side:     side,
			Position: &schema.Position{Lat: lat, Lon: lon, AltM: 5000},
		})
		s.Missions = append(s.Missions, &schema.EntityMission{
			EntityId: id,
			Roe:      schema.ROE_ROE_WEAPONS_FREE,
			Waypoints: []*schema.Waypoint{
				{Lat: 20.0 + r.Float64()*10, Lon: 50.0 + r.Float64()*10, AltM: 5000, SpeedMs: 200},
			},
		})
	}
	return s
}

func benchmarkEngine(b *testing.B, n int) {
	s := genScenario(n, 42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newEngine(s, "bench-run").run("bench", engineID, "bench-run")
	}
}

func BenchmarkEngine100(b *testing.B)  { benchmarkEngine(b, 100) }
func BenchmarkEngine400(b *testing.B)  { benchmarkEngine(b, 400) }
func BenchmarkEngine1000(b *testing.B) { benchmarkEngine(b, 1000) }

// TestGridMatchesBruteForce is the safety net for the spatial broadphase: for a
// scenario large enough to trigger the grid, the grid and brute-force paths
// must produce byte-identical results (same events, kills, and tracks). Any
// divergence in the RNG draw order would surface here.
func TestGridMatchesBruteForce(t *testing.T) {
	s := genScenario(200, 7)

	grid := newEngine(s, "eq-run")
	if !grid.useGrid {
		t.Fatal("expected the grid broadphase to be enabled for a 200-entity scenario")
	}
	brute := newEngine(s, "eq-run")
	brute.useGrid = false

	rg := grid.run("bench", engineID, "eq-run")
	rb := brute.run("bench", engineID, "eq-run")

	if len(rg.Events) != len(rb.Events) {
		t.Fatalf("event count differs: grid=%d brute=%d", len(rg.Events), len(rb.Events))
	}
	for i := range rg.Events {
		g, b := rg.Events[i], rb.Events[i]
		if g.Type != b.Type || g.EntityId != b.EntityId || g.TargetEntityId != b.TargetEntityId ||
			g.Detail != b.Detail || g.Ts.GetSeconds() != b.Ts.GetSeconds() {
			t.Fatalf("event %d differs:\n grid=%+v\nbrute=%+v", i, g, b)
		}
	}

	if len(rg.KillChains) != len(rb.KillChains) {
		t.Fatalf("kill-chain count differs: grid=%d brute=%d", len(rg.KillChains), len(rb.KillChains))
	}
	for i := range rg.KillChains {
		if rg.KillChains[i].AttackerEntityId != rb.KillChains[i].AttackerEntityId ||
			rg.KillChains[i].TargetEntityId != rb.KillChains[i].TargetEntityId {
			t.Fatalf("kill chain %d differs: grid=%+v brute=%+v", i, rg.KillChains[i], rb.KillChains[i])
		}
	}

	if len(rg.EntityTracks) != len(rb.EntityTracks) {
		t.Fatalf("track count differs: grid=%d brute=%d", len(rg.EntityTracks), len(rb.EntityTracks))
	}
	for ti := range rg.EntityTracks {
		gt, bt := rg.EntityTracks[ti], rb.EntityTracks[ti]
		if gt.EntityId != bt.EntityId || len(gt.Points) != len(bt.Points) {
			t.Fatalf("track %d shape differs", ti)
		}
		for pi := range gt.Points {
			gp, bp := gt.Points[pi], bt.Points[pi]
			if gp.Lat != bp.Lat || gp.Lon != bp.Lon || gp.Status != bp.Status ||
				gp.HeadingDeg != bp.HeadingDeg || gp.SpeedMs != bp.SpeedMs {
				t.Fatalf("track %s point %d differs:\n grid=%+v\nbrute=%+v", gt.EntityId, pi, gp, bp)
			}
		}
	}
}
