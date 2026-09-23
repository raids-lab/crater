package extender

import (
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/pkg/utils"
)

// reservationTTL is only a backstop. Entries normally leave through sweep once the snapshot shows the
// job admitted, terminal or gone. Every pass through decide refreshes the timestamp, so a job that
// capacity keeps rejecting for now but may still admit later holds its reservation for as long as
// volcano keeps asking about it: that is the intended first-come-first-served order. The TTL covers
// the remaining case where the extender is no longer asked at all, e.g. an earlier-tier plugin
// started rejecting the job. Jobs that can never be admitted are kept out of the ledger by decide's
// capability guard.
const reservationTTL = 30 * time.Second

type reservation struct {
	userID    uint
	queue     string
	resources v1.ResourceList
	at        time.Time
}

// sessionAccumulator tracks jobs this process let through but whose admission volcano writes back
// only at session close. Without it a batch submission is measured against one stale usage snapshot
// and the whole batch passes the same quota check.
type sessionAccumulator struct {
	mu      sync.Mutex
	entries map[string]*reservation
}

func newSessionAccumulator() *sessionAccumulator {
	return &sessionAccumulator{entries: make(map[string]*reservation)}
}

func (a *sessionAccumulator) reserve(view *jobView) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries[view.name] = &reservation{
		userID:    view.userID,
		queue:     view.queue,
		resources: view.resources,
		at:        utils.GetLocalTime(),
	}
}

// sweep releases a reservation as soon as the pod group cache confirms the admission it stood in for,
// which is the same condition that makes the job show up in the admitted usage sum.
func (a *sessionAccumulator) sweep(snap *snapshot) {
	deadline := utils.GetLocalTime().Add(-reservationTTL)
	a.mu.Lock()
	defer a.mu.Unlock()
	for name, entry := range a.entries {
		view, ok := snap.byName[name]
		if !ok || utils.IsPodGroupAdmitted(view.podGroupPhase) ||
			utils.IsJobPhaseTerminal(view.jobPhase) || entry.at.Before(deadline) {
			delete(a.entries, name)
		}
	}
}

// reservedExcluding skips the inspected job's own standing reservation, which would otherwise be
// counted twice against it when volcano asks about the same job in a later session.
func (a *sessionAccumulator) reservedExcluding(userID uint, queue, jobName string) v1.ResourceList {
	a.mu.Lock()
	defer a.mu.Unlock()
	total := v1.ResourceList{}
	for name, entry := range a.entries {
		if name == jobName || entry.userID != userID || entry.queue != queue {
			continue
		}
		total = utils.SumResources(total, entry.resources)
	}
	return total
}
