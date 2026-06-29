package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"time"

	"github.com/usip/backend/schema"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	earthRadiusM  = 6371000.0
	defaultStepS  = 1.0
	reengageCoolS = 10.0 // minimum seconds between shots at the same target
	fallbackSpeed = 100.0
)

// engine holds simulation state and result accumulators for a single run.
type engine struct {
	entities  []*simEntity
	startTime time.Time
	stepS     float64
	durationS float64
	rng       *rand.Rand

	tracks     map[string]*schema.EntityTrack
	trackOrder []string // preserve entity order for deterministic output
	events     []*schema.SimEvent
	killChains []*schema.KillChain
	lastEngage map[string]float64 // "attacker|target" -> last engagement sim-time
}

func newEngine(s *schema.ScenarioProto, runID string) *engine {
	step := defaultStepS
	dur := float64(s.DurationS)
	if dur <= 0 {
		dur = 600 // default 10 minutes of sim time
	}
	start := time.Now().UTC()
	if s.StartTime != nil {
		start = s.StartTime.AsTime()
	}

	ents := translate(s)

	e := &engine{
		entities:   ents,
		startTime:  start,
		stepS:      step,
		durationS:  dur,
		rng:        rand.New(rand.NewSource(seedFromRunID(runID))),
		tracks:     make(map[string]*schema.EntityTrack, len(ents)),
		lastEngage: make(map[string]float64),
	}
	for _, ent := range ents {
		e.tracks[ent.id] = &schema.EntityTrack{EntityId: ent.id}
		e.trackOrder = append(e.trackOrder, ent.id)
	}
	return e
}

// run executes the full simulation and returns normalized results.
func (e *engine) run(scenarioID, engineID, runID string) *schema.SimResultsProto {
	for t := 0.0; t <= e.durationS; t += e.stepS {
		for _, ent := range e.entities {
			if ent.alive {
				e.advance(ent, t)
			}
			e.recordTrack(ent, t)
		}
		e.interactions(t)
	}

	return &schema.SimResultsProto{
		ScenarioId:   scenarioID,
		EngineId:     engineID,
		RunId:        runID,
		EntityTracks: e.orderedTracks(),
		Events:       e.events,
		KillChains:   e.killChains,
		MoeMetrics:   e.computeMOEs(),
	}
}

// advance moves an entity along its waypoints for one time step.
func (e *engine) advance(ent *simEntity, t float64) {
	if ent.wpIndex >= len(ent.waypoints) {
		ent.speedMS = 0
		return
	}
	if t < ent.holdUntilS {
		ent.speedMS = 0
		return
	}

	target := ent.waypoints[ent.wpIndex]
	speed := target.speedMS
	if speed <= 0 {
		speed = fallbackSpeed
	}

	dist := haversineM(ent.lat, ent.lon, target.lat, target.lon)
	stepDist := speed * e.stepS

	if stepDist >= dist {
		// Reached the waypoint this step.
		ent.lat, ent.lon, ent.alt = target.lat, target.lon, target.alt
		ent.speedMS = speed
		e.emit(t, schema.EventType_EVENT_TYPE_WAYPOINT_REACHED, ent.id, "",
			fmt.Sprintf("reached waypoint %d", ent.wpIndex))
		if target.holdTimeS > 0 {
			ent.holdUntilS = t + float64(target.holdTimeS)
		}
		ent.wpIndex++
		if ent.wpIndex >= len(ent.waypoints) {
			e.emit(t, schema.EventType_EVENT_TYPE_MISSION_COMPLETE, ent.id, "",
				"final waypoint reached")
		}
		return
	}

	brng := bearingDeg(ent.lat, ent.lon, target.lat, target.lon)
	ent.lat, ent.lon = movePoint(ent.lat, ent.lon, stepDist, brng)
	ent.alt = target.alt
	ent.headingDeg = brng
	ent.speedMS = speed
}

// interactions runs detection, engagement, and kill resolution for one step.
func (e *engine) interactions(t float64) {
	for _, a := range e.entities {
		if !a.alive || a.sensorRangeM <= 0 {
			continue
		}
		for _, b := range e.entities {
			if b == a || !b.alive || !areEnemies(a.side, b.side) {
				continue
			}
			d := haversineM(a.lat, a.lon, b.lat, b.lon)

			// Detection
			if d <= a.sensorRangeM && !a.detected[b.id] {
				a.detected[b.id] = true
				e.emit(t, schema.EventType_EVENT_TYPE_DETECTION, a.id, b.id,
					fmt.Sprintf("detected %s at %.0f m", b.name, d))
			}

			// Engagement + kill
			if a.weaponRangeM > 0 && roeAllows(a) && d <= a.weaponRangeM {
				key := a.id + "|" + b.id
				if t-e.lastEngage[key] < reengageCoolS && e.lastEngage[key] != 0 {
					continue
				}
				e.lastEngage[key] = t
				e.emit(t, schema.EventType_EVENT_TYPE_ENGAGEMENT, a.id, b.id,
					fmt.Sprintf("engaging %s at %.0f m", b.name, d))

				pk := capabilityFor(a.etype).basePk
				if e.rng.Float64() < pk {
					b.alive = false
					e.emit(t, schema.EventType_EVENT_TYPE_KILL, a.id, b.id,
						fmt.Sprintf("killed %s", b.name))
					ts := e.ts(t)
					e.killChains = append(e.killChains, &schema.KillChain{
						AttackerEntityId: a.id,
						TargetEntityId:   b.id,
						EngagedAt:        ts,
						KilledAt:         ts,
					})
				}
			}
		}
	}
}

func (e *engine) recordTrack(ent *simEntity, t float64) {
	status := "alive"
	if !ent.alive {
		status = "killed"
	}
	e.tracks[ent.id].Points = append(e.tracks[ent.id].Points, &schema.TrackPoint{
		Ts:         e.ts(t),
		Lat:        ent.lat,
		Lon:        ent.lon,
		AltM:       ent.alt,
		HeadingDeg: ent.headingDeg,
		SpeedMs:    ent.speedMS,
		Status:     status,
	})
}

func (e *engine) computeMOEs() []*schema.MOEMetric {
	var blueLosses, redLosses, blueKills, redKills float64
	for _, ent := range e.entities {
		if !ent.alive {
			switch ent.side {
			case schema.Side_SIDE_FRIENDLY:
				blueLosses++
			case schema.Side_SIDE_ENEMY:
				redLosses++
			}
		}
	}
	for _, kc := range e.killChains {
		attacker := e.entityByID(kc.AttackerEntityId)
		if attacker == nil {
			continue
		}
		switch attacker.side {
		case schema.Side_SIDE_FRIENDLY:
			blueKills++
		case schema.Side_SIDE_ENEMY:
			redKills++
		}
	}
	return []*schema.MOEMetric{
		{Key: "blue_losses", Value: blueLosses, Unit: "entities"},
		{Key: "red_losses", Value: redLosses, Unit: "entities"},
		{Key: "blue_kills", Value: blueKills, Unit: "entities"},
		{Key: "red_kills", Value: redKills, Unit: "entities"},
		{Key: "total_kills", Value: float64(len(e.killChains)), Unit: "entities"},
	}
}

// --- helpers ---

func (e *engine) emit(t float64, typ schema.EventType, entityID, targetID, detail string) {
	e.events = append(e.events, &schema.SimEvent{
		Ts:             e.ts(t),
		Type:           typ,
		EntityId:       entityID,
		TargetEntityId: targetID,
		Detail:         detail,
	})
}

func (e *engine) ts(t float64) *timestamppb.Timestamp {
	return timestamppb.New(e.startTime.Add(time.Duration(t * float64(time.Second))))
}

func (e *engine) orderedTracks() []*schema.EntityTrack {
	out := make([]*schema.EntityTrack, 0, len(e.trackOrder))
	for _, id := range e.trackOrder {
		out = append(out, e.tracks[id])
	}
	return out
}

func (e *engine) entityByID(id string) *simEntity {
	for _, ent := range e.entities {
		if ent.id == id {
			return ent
		}
	}
	return nil
}

func areEnemies(a, b schema.Side) bool {
	return (a == schema.Side_SIDE_FRIENDLY && b == schema.Side_SIDE_ENEMY) ||
		(a == schema.Side_SIDE_ENEMY && b == schema.Side_SIDE_FRIENDLY)
}

func roeAllows(a *simEntity) bool {
	switch a.roe {
	case schema.ROE_ROE_WEAPONS_TIGHT, schema.ROE_ROE_WEAPONS_FREE:
		return true
	default: // WEAPONS_HOLD or unspecified
		return false
	}
}

func seedFromRunID(runID string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(runID))
	return int64(h.Sum64())
}

// haversineM returns great-circle distance in meters between two lat/lon points.
func haversineM(lat1, lon1, lat2, lon2 float64) float64 {
	p1, p2 := rad(lat1), rad(lat2)
	dLat, dLon := rad(lat2-lat1), rad(lon2-lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(p1)*math.Cos(p2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// bearingDeg returns the initial bearing (degrees) from point 1 to point 2.
func bearingDeg(lat1, lon1, lat2, lon2 float64) float64 {
	p1, p2 := rad(lat1), rad(lat2)
	dLon := rad(lon2 - lon1)
	y := math.Sin(dLon) * math.Cos(p2)
	x := math.Cos(p1)*math.Sin(p2) - math.Sin(p1)*math.Cos(p2)*math.Cos(dLon)
	return math.Mod(deg(math.Atan2(y, x))+360, 360)
}

// movePoint advances a lat/lon point by distance d (meters) along a bearing.
func movePoint(lat, lon, d, bearing float64) (float64, float64) {
	ad := d / earthRadiusM
	br := rad(bearing)
	p1 := rad(lat)
	l1 := rad(lon)
	p2 := math.Asin(math.Sin(p1)*math.Cos(ad) + math.Cos(p1)*math.Sin(ad)*math.Cos(br))
	l2 := l1 + math.Atan2(
		math.Sin(br)*math.Sin(ad)*math.Cos(p1),
		math.Cos(ad)-math.Sin(p1)*math.Sin(p2),
	)
	return deg(p2), deg(l2)
}

func rad(d float64) float64 { return d * math.Pi / 180 }
func deg(r float64) float64 { return r * 180 / math.Pi }
