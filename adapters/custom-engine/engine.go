package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
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

	// Spatial broadphase: for large scenarios the O(n^2) pair loop dominates,
	// so we cull far pairs with a uniform lat/lon grid. Enabled only when it's
	// both worthwhile and safe (see shouldUseGrid); results are identical to
	// the brute-force path (proven by TestGridMatchesBruteForce).
	useGrid    bool
	gridLatDeg float64 // cell height in degrees latitude
	gridLonDeg float64 // cell width in degrees longitude
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
	e.configureGrid()
	return e
}

// gridThreshold is the entity count above which the spatial grid pays for its
// per-step build overhead. Below it the brute-force loop is faster.
const gridThreshold = 64

// configureGrid decides whether to use the spatial broadphase and, if so,
// computes cell sizes. It falls back to brute force for small scenarios and
// for geometries where a uniform lat/lon grid is unsafe (antimeridian span or
// near-polar latitudes, where degrees-of-longitude scaling breaks down).
func (e *engine) configureGrid() {
	if len(e.entities) < gridThreshold {
		return
	}
	// maxR is the largest interaction radius any entity has; a cell sized to
	// it guarantees every in-range pair lands in the same or an adjacent cell.
	var maxR, maxAbsLat, minLon, maxLon float64
	minLon, maxLon = 180, -180
	for _, ent := range e.entities {
		if r := math.Max(ent.sensorRangeM, ent.weaponRangeM); r > maxR {
			maxR = r
		}
		if math.Abs(ent.lat) > maxAbsLat {
			maxAbsLat = math.Abs(ent.lat)
		}
		if ent.lon < minLon {
			minLon = ent.lon
		}
		if ent.lon > maxLon {
			maxLon = ent.lon
		}
		for _, w := range ent.waypoints {
			if math.Abs(w.lat) > maxAbsLat {
				maxAbsLat = math.Abs(w.lat)
			}
			if w.lon < minLon {
				minLon = w.lon
			}
			if w.lon > maxLon {
				maxLon = w.lon
			}
		}
	}
	if maxR <= 0 || maxAbsLat > 85 || maxLon-minLon > 180 {
		return // unsafe or pointless; keep brute force
	}
	// Degrees per meter: latitude is ~constant; longitude shrinks with cos(lat),
	// so size cells using the highest latitude present (smallest cos) to
	// guarantee every cell is at least maxR wide everywhere in the scenario.
	metersPerDegLat := earthRadiusM * math.Pi / 180
	metersPerDegLon := metersPerDegLat * math.Cos(maxAbsLat*math.Pi/180)
	if metersPerDegLon <= 0 {
		return
	}
	e.gridLatDeg = maxR / metersPerDegLat
	e.gridLonDeg = maxR / metersPerDegLon
	e.useGrid = true
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

// interactions runs detection, engagement, and kill resolution for one step,
// dispatching to the brute-force or grid broadphase. Both visit attackers in
// entity order and, per attacker, candidate targets in ascending entity index
// order, so the RNG draw sequence — and thus the result — is identical.
func (e *engine) interactions(t float64) {
	if e.useGrid {
		e.interactionsGrid(t)
	} else {
		e.interactionsBrute(t)
	}
}

func (e *engine) interactionsBrute(t float64) {
	for _, a := range e.entities {
		if !a.alive || a.sensorRangeM <= 0 {
			continue
		}
		for _, b := range e.entities {
			e.interact(t, a, b)
		}
	}
}

// interactionsGrid is the broadphase: each attacker only tests targets in its
// own and adjacent grid cells. Since cells are sized to the largest interaction
// radius, no in-range pair is ever missed; candidates are visited in ascending
// entity index order to match the brute-force RNG sequence exactly.
func (e *engine) interactionsGrid(t float64) {
	cells := e.buildGrid()
	for _, a := range e.entities {
		if !a.alive || a.sensorRangeM <= 0 {
			continue
		}
		for _, bi := range e.candidates(cells, a) {
			e.interact(t, a, e.entities[bi])
		}
	}
}

// interact resolves detection/engagement/kill for a single ordered pair (a
// attacking b). It is the single source of truth shared by both broadphases.
func (e *engine) interact(t float64, a, b *simEntity) {
	if b == a || !b.alive || !areEnemies(a.side, b.side) {
		return
	}
	d := haversineM(a.lat, a.lon, b.lat, b.lon)

	// Detection
	if d <= a.sensorRangeM && !a.detected[b.id] && e.rollDetection(a, d) {
		a.detected[b.id] = true
		e.emit(t, schema.EventType_EVENT_TYPE_DETECTION, a.id, b.id,
			fmt.Sprintf("detected %s at %.0f m", b.name, d))
	}

	// Engagement + kill
	if a.weaponRangeM > 0 && a.ammo != 0 && roeAllows(a) && d <= a.weaponRangeM {
		key := a.id + "|" + b.id
		if t-e.lastEngage[key] < reengageCoolS && e.lastEngage[key] != 0 {
			return
		}
		e.lastEngage[key] = t
		if a.ammo > 0 {
			a.ammo--
		}
		e.emit(t, schema.EventType_EVENT_TYPE_ENGAGEMENT, a.id, b.id,
			fmt.Sprintf("engaging %s at %.0f m", b.name, d))

		if e.rng.Float64() < a.basePk {
			b.healthHP--
			if b.healthHP > 0 {
				e.emit(t, schema.EventType_EVENT_TYPE_DAMAGE, a.id, b.id,
					fmt.Sprintf("damaged %s (%.0f/%.0f HP)", b.name, b.healthHP, b.healthMaxHP))
				return
			}
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

// gridCell keys a spatial bucket.
type gridCell struct{ lat, lon int }

// buildGrid buckets currently-alive entities into cells by position, in
// ascending entity-index order (so each bucket is pre-sorted).
func (e *engine) buildGrid() map[gridCell][]int {
	cells := make(map[gridCell][]int, len(e.entities))
	for i, ent := range e.entities {
		if !ent.alive {
			continue
		}
		c := e.cellOf(ent)
		cells[c] = append(cells[c], i)
	}
	return cells
}

func (e *engine) cellOf(ent *simEntity) gridCell {
	return gridCell{
		lat: int(math.Floor(ent.lat / e.gridLatDeg)),
		lon: int(math.Floor(ent.lon / e.gridLonDeg)),
	}
}

// candidates returns the entity indices in a's own and adjacent cells, sorted
// ascending to preserve brute-force iteration order.
func (e *engine) candidates(cells map[gridCell][]int, a *simEntity) []int {
	c := e.cellOf(a)
	var out []int
	for dlat := -1; dlat <= 1; dlat++ {
		for dlon := -1; dlon <= 1; dlon++ {
			out = append(out, cells[gridCell{c.lat + dlat, c.lon + dlon}]...)
		}
	}
	sort.Ints(out)
	return out
}

// rollDetection returns whether entity a detects a target at distance d
// (already confirmed <= a.sensorRangeM by the caller). Entities without
// pdFalloff always detect (hard cutoff, the engine's original behavior).
func (e *engine) rollDetection(a *simEntity, d float64) bool {
	if !a.pdFalloff {
		return true
	}
	pd := lerp(1.0, a.pdMin, d/a.sensorRangeM)
	return e.rng.Float64() < pd
}

func (e *engine) recordTrack(ent *simEntity, t float64) {
	status := "alive"
	if !ent.alive {
		status = "killed"
	} else if ent.healthHP < ent.healthMaxHP {
		status = "damaged"
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

// computeMOEs returns the canonical cross-engine MOE set documented in
// docs/moe-taxonomy.md, plus any engine-specific metrics.
func (e *engine) computeMOEs() []*schema.MOEMetric {
	var blueLosses, redLosses, blueKills, redKills float64
	var healthPctSum float64
	for _, ent := range e.entities {
		if !ent.alive {
			switch ent.side {
			case schema.Side_SIDE_FRIENDLY:
				blueLosses++
			case schema.Side_SIDE_ENEMY:
				redLosses++
			}
		}
		healthPctSum += 100 * math.Max(ent.healthHP, 0) / ent.healthMaxHP
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

	var detections, engagements float64
	for _, ev := range e.events {
		switch ev.Type {
		case schema.EventType_EVENT_TYPE_DETECTION:
			detections++
		case schema.EventType_EVENT_TYPE_ENGAGEMENT:
			engagements++
		}
	}

	avgHealthPct := 0.0
	if len(e.entities) > 0 {
		avgHealthPct = healthPctSum / float64(len(e.entities))
	}

	return []*schema.MOEMetric{
		{Key: "blue_losses", Value: blueLosses, Unit: "entities"},
		{Key: "red_losses", Value: redLosses, Unit: "entities"},
		{Key: "blue_kills", Value: blueKills, Unit: "entities"},
		{Key: "red_kills", Value: redKills, Unit: "entities"},
		{Key: "total_kills", Value: float64(len(e.killChains)), Unit: "entities"},
		{Key: "detections_total", Value: detections, Unit: "events"},
		{Key: "rounds_expended", Value: engagements, Unit: "rounds"},
		{Key: "avg_health_pct", Value: avgHealthPct, Unit: "percent"},
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

// lerp linearly interpolates between a and b as frac goes from 0 to 1,
// clamping frac to [0, 1].
func lerp(a, b, frac float64) float64 {
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	return a + (b-a)*frac
}
