package monitor

import (
	"container/heap"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// Match Prometheus's default lookback for held gauges between observations.
const memoryLookback = 5 * time.Minute

type memoryReading struct {
	prompb.Sample
	valid bool
}

func (r memoryReading) expiry() int64 {
	return r.Timestamp + memoryLookback.Milliseconds() + 1
}

type memoryDevice struct {
	used, free memoryReading
}

func (d *memoryDevice) totals() (used, capacity float64, active, paired int) {
	if d.used.valid {
		used = d.used.Value
	}
	if d.used.valid || d.free.valid {
		active = 1
	}
	// A capacity observation requires used and free from the same scrape.
	if d.used.valid && d.free.valid && d.used.Timestamp == d.free.Timestamp {
		capacity, paired = used+d.free.Value, 1
	}
	return used, capacity, active, paired
}

type memoryTimeline struct {
	devices        map[string]*memoryDevice
	gpu            bool
	active, paired int
	used, capacity float64
	usedDevices    int
	usageStats     metricMoments
	capacityStats  metricMoments
}

type memoryCursor struct {
	device  *memoryDevice
	reading *memoryReading
	samples []prompb.Sample
	at      int64
	expire  bool
}

func (c *memoryCursor) schedule() bool {
	c.expire = c.reading.valid
	if c.expire {
		c.at = c.reading.expiry()
	}
	if len(c.samples) > 0 && (!c.expire || c.samples[0].Timestamp <= c.at) {
		c.at, c.expire = c.samples[0].Timestamp, false
		return true
	}
	return c.expire
}

func (c *memoryCursor) advance() bool {
	if c.expire {
		c.reading.valid = false
		return false
	}
	sample := c.samples[0]
	c.samples = c.samples[1:]
	*c.reading = memoryReading{Sample: sample, valid: isFinite(sample.Value) && sample.Value >= 0}
	return c.reading.valid
}

type memoryEvents []*memoryCursor

func (h memoryEvents) Len() int           { return len(h) }
func (h memoryEvents) Less(i, j int) bool { return h[i].at < h[j].at }
func (h memoryEvents) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *memoryEvents) Push(x any)        { *h = append(*h, x.(*memoryCursor)) }
func (h *memoryEvents) Pop() any {
	old := *h
	n := len(old) - 1
	x := old[n]
	old[n] = nil
	*h = old[:n]
	return x
}

func (m *memoryTimeline) chunkEvents(series []*prompb.TimeSeries) memoryEvents {
	byReading := make(map[*memoryReading][]prompb.Sample, len(series))
	for _, stream := range series {
		key := profileSeriesKey(stream, m.gpu)
		device := m.devices[key]
		if device == nil {
			device = &memoryDevice{}
			m.devices[key] = device
		}
		reading := &device.used
		if m.gpu && profileMetricName(stream) == gpuFBFree {
			reading = &device.free
		}
		byReading[reading] = stream.Samples
	}

	events := make(memoryEvents, 0, len(byReading))
	for _, device := range m.devices {
		for _, reading := range []*memoryReading{&device.used, &device.free} {
			cursor := &memoryCursor{device: device, reading: reading, samples: byReading[reading]}
			if cursor.schedule() {
				events = append(events, cursor)
			}
		}
	}
	return events
}

func (m *memoryTimeline) consume(series []*prompb.TimeSeries, end int64) error {
	events := m.chunkEvents(series)
	heap.Init(&events)
	for events.Len() > 0 && events[0].at <= end {
		observedUsed, observedGPU := m.advanceTimestamp(&events, events[0].at)
		if observedUsed {
			m.usageStats.add(m.used, 1)
		}
		if m.gpu && observedGPU && m.active > 0 && m.active == m.paired {
			m.capacityStats.add(m.capacity, 1)
		}
	}
	for key, device := range m.devices {
		if !device.used.valid && !device.free.valid {
			delete(m.devices, key)
		}
	}
	return nil
}

// Apply all observations at the same timestamp before recording a total.
func (m *memoryTimeline) advanceTimestamp(events *memoryEvents, at int64) (observedUsed, observedGPU bool) {
	for events.Len() > 0 && (*events)[0].at == at {
		cursor := heap.Pop(events).(*memoryCursor)
		observed := m.advance(cursor)
		observedUsed = observedUsed || (observed && cursor.reading == &cursor.device.used)
		observedGPU = observedGPU || observed
		if cursor.schedule() {
			heap.Push(events, cursor)
		}
	}
	return observedUsed, observedGPU
}

func (m *memoryTimeline) advance(cursor *memoryCursor) bool {
	usedBefore, capacityBefore, activeBefore, pairedBefore := cursor.device.totals()
	if cursor.device.used.valid {
		m.usedDevices--
	}
	observed := cursor.advance()
	if cursor.device.used.valid {
		m.usedDevices++
	}
	used, capacity, active, paired := cursor.device.totals()
	m.used += used - usedBefore
	m.capacity += capacity - capacityBefore
	m.active += active - activeBefore
	m.paired += paired - pairedBefore
	if m.usedDevices == 0 {
		m.used = 0
	}
	if m.paired == 0 {
		m.capacity = 0
	}
	return observed
}

func (m *memoryTimeline) summary() (used, capacity *metricSummary) {
	used, capacity = m.usageStats.summary(), m.capacityStats.summary()
	if used != nil && capacity != nil && capacity.max < used.max {
		// A partial capacity observation cannot explain the measured usage peak.
		capacity = nil
	}
	return used, capacity
}
