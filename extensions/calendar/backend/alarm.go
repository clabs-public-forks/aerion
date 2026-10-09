package backend

// VALARM parser + per-instance trigger computation. Phase 1G.
//
// ExtractAlarms walks the VALARM components nested inside an Event's
// stored ICSBlob and projects them onto the event's expanded instances,
// producing one Alarm per (occurrence × VALARM). RECURRENCE-ID overrides
// contribute their own VALARMs when present (override > master).
//
// TRIGGER encoding per RFC 5545 §3.8.6.3:
//   - Duration form (default): "-PT15M" → 15 minutes before related point.
//   - RELATED=START (default) → relative to instance DTSTART.
//   - RELATED=END → relative to instance DTEND.
//   - VALUE=DATE-TIME form: absolute UTC instant (rare; uses VEVENT-tz
//     conversion if TZID present).
//
// ACTION: only "DISPLAY" is dispatched today. Others are stored on the
// Alarm so the scheduler can filter; future Phase 2+ can route AUDIO /
// EMAIL / PROCEDURE through different mechanisms.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/google/uuid"
)

// ExtractAlarms computes Alarm rows for one Event + its expanded
// instances. Caller passes the master event (with ICSBlob) and the list of
// EventOverrides whose ics_blob may include their own VALARM blocks.
//
// The returned alarms have ID auto-generated, Status="pending", and
// CreatedAt=0 (filled in by Store.ReplacePendingAlarmsTx).
func ExtractAlarms(ev Event, overrides []EventOverride, instances []EventInstance) ([]Alarm, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	masterAlarms, err := parseAlarmTemplates(ev.ICSBlob)
	if err != nil {
		return nil, fmt.Errorf("parse master alarms: %w", err)
	}

	// Override → per-instance templates (keyed by RecurrenceIDUnix).
	overrideTemplates := make(map[int64][]alarmTemplate, len(overrides))
	for _, ov := range overrides {
		tmpls, err := parseAlarmTemplates(ov.ICSBlob)
		if err != nil {
			// One bad override shouldn't kill the whole pipeline.
			continue
		}
		overrideTemplates[ov.RecurrenceIDUnix] = tmpls
	}

	out := make([]Alarm, 0, len(instances))
	for _, inst := range instances {
		templates := masterAlarms
		if ov, ok := overrideTemplates[inst.RecurrenceIDUnix]; ok {
			// Override takes precedence; if the override has zero alarms,
			// it means the user intentionally cleared reminders for that
			// instance — honor that.
			templates = ov
		}
		for _, t := range templates {
			triggerUnix := computeTriggerUnix(t, inst)
			out = append(out, Alarm{
				ID:           uuid.NewString(),
				EventID:      ev.ID,
				InstanceUnix: inst.InstanceStartUnix,
				TriggerUnix:  triggerUnix,
				Action:       strings.ToLower(t.action),
				Description:  t.description,
			})
		}
	}
	return out, nil
}

// alarmWindow is how far ahead alarms are materialized into event_alarms.
// It must exceed the scheduler's arming horizon, and the hourly refresh
// rolls it forward so recurring events always have upcoming alarms.
const alarmWindow = alarmHorizon + 8*24*time.Hour

// upcomingAlarms expands ev over the alarm window and returns the alarms
// that trigger at or after now. Expansion starts one horizon back so an
// occurrence that already started can still contribute a trigger relative
// to its end or a positive offset.
func upcomingAlarms(ev Event, overrides []EventOverride, now time.Time) ([]Alarm, error) {
	instances, err := ExpandInRange(ev, overrides, now.Add(-alarmHorizon), now.Add(alarmWindow))
	if err != nil {
		return nil, fmt.Errorf("expand for alarms: %w", err)
	}
	alarms, err := ExtractAlarms(ev, overrides, instances)
	if err != nil {
		return nil, fmt.Errorf("extract alarms: %w", err)
	}
	upcoming := alarms[:0]
	for _, a := range alarms {
		if a.TriggerUnix >= now.Unix() {
			upcoming = append(upcoming, a)
		}
	}
	return upcoming, nil
}

// refreshEventAlarmsTx replaces ev's future pending alarms with the ones its
// current data produces, dropping alarms left behind by an edit.
func refreshEventAlarmsTx(tx *sql.Tx, store *Store, ev Event, overrides []EventOverride, now time.Time) error {
	alarms, err := upcomingAlarms(ev, overrides, now)
	if err != nil {
		return err
	}
	return store.ReplacePendingAlarmsTx(tx, ev.ID, now.Unix(), alarms)
}

// RefreshAllAlarms recomputes the future pending alarms of every event in
// one transaction. Events that fail to expand are skipped and reported in
// the returned error; the others are still refreshed. Cancelling ctx stops
// the pass before anything is written.
func RefreshAllAlarms(ctx context.Context, store *Store, now time.Time) error {
	calendarIDs, err := store.ListCalendarIDs()
	if err != nil {
		return err
	}
	return refreshCalendarAlarms(ctx, store, calendarIDs, now)
}

// RefreshSourceAlarms is RefreshAllAlarms limited to one source's events,
// used after that source syncs.
func RefreshSourceAlarms(ctx context.Context, store *Store, sourceID string, now time.Time) error {
	calendarIDs, err := store.ListCalendarIDsForSource(sourceID)
	if err != nil {
		return err
	}
	return refreshCalendarAlarms(ctx, store, calendarIDs, now)
}

// refreshCalendarAlarms recomputes the future pending alarms of every event
// in calendarIDs, loading all their overrides in one query.
func refreshCalendarAlarms(ctx context.Context, store *Store, calendarIDs []string, now time.Time) error {
	if len(calendarIDs) == 0 {
		return nil
	}
	events, err := store.ListEventsForExpansion(calendarIDs)
	if err != nil {
		return err
	}
	overridesByEvent, err := store.ListOverridesForCalendars(calendarIDs)
	if err != nil {
		return err
	}

	type eventAlarms struct {
		eventID string
		alarms  []Alarm
	}
	planned := make([]eventAlarms, 0, len(events))
	var expandErrs []error
	for _, ev := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		alarms, err := upcomingAlarms(ev, overridesByEvent[ev.ID], now)
		if err != nil {
			expandErrs = append(expandErrs, fmt.Errorf("event %s: %w", ev.ID, err))
			continue
		}
		planned = append(planned, eventAlarms{ev.ID, alarms})
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	err = store.WithTx(func(tx *sql.Tx) error {
		for _, p := range planned {
			if err := store.ReplacePendingAlarmsTx(tx, p.eventID, now.Unix(), p.alarms); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return errors.Join(expandErrs...)
}

// alarmTemplate is one VALARM parsed from an ICSBlob — not yet projected
// onto a concrete instance. Holds the raw TRIGGER + ACTION + DESCRIPTION
// so the per-instance projection step is a simple arithmetic application.
type alarmTemplate struct {
	action      string // DISPLAY / AUDIO / EMAIL / PROCEDURE (raw casing)
	description string

	// One of the two trigger encodings is populated:
	relativeSeconds int64 // offset in seconds; negative = before reference
	relatedToEnd    bool  // RELATED=END instead of START
	absoluteUnix    int64 // if non-zero, this is an absolute trigger
}

// parseAlarmTemplates decodes one ICSBlob (a VCALENDAR wrapping VEVENTs)
// and returns the VALARM templates of the first VEVENT that doesn't
// have a RECURRENCE-ID (the master); overrides should call this on their
// own ICSBlob which holds exactly one VEVENT.
func parseAlarmTemplates(icsBlob string) ([]alarmTemplate, error) {
	if icsBlob == "" {
		return nil, nil
	}
	cal, err := decodeICS(icsBlob)
	if err != nil {
		return nil, fmt.Errorf("decode ics: %w", err)
	}
	events := cal.Events()
	if len(events) == 0 {
		return nil, nil
	}

	// Master = first VEVENT without RECURRENCE-ID, else first.
	var ev *ical.Event
	for i := range events {
		if events[i].Props.Get(ical.PropRecurrenceID) == nil {
			e := events[i]
			ev = &e
			break
		}
	}
	if ev == nil {
		first := events[0]
		ev = &first
	}

	out := make([]alarmTemplate, 0, len(ev.Component.Children))
	for _, child := range ev.Component.Children {
		if child.Name != ical.CompAlarm {
			continue
		}
		t, ok := buildAlarmTemplate(child)
		if !ok {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func buildAlarmTemplate(comp *ical.Component) (alarmTemplate, bool) {
	t := alarmTemplate{action: "DISPLAY"}

	if actionProp := comp.Props.Get(ical.PropAction); actionProp != nil {
		t.action = strings.ToUpper(strings.TrimSpace(actionProp.Value))
	}
	if descProp := comp.Props.Get(ical.PropDescription); descProp != nil {
		t.description = descProp.Value
	}

	triggerProp := comp.Props.Get(ical.PropTrigger)
	if triggerProp == nil {
		return t, false
	}

	// Detect absolute vs relative trigger. RFC 5545: absolute requires
	// VALUE=DATE-TIME; otherwise default is DURATION.
	valueType := strings.ToUpper(strings.TrimSpace(triggerProp.Params.Get(ical.ParamValue)))
	if valueType == "DATE-TIME" {
		dt, err := triggerProp.DateTime(nil)
		if err != nil {
			return t, false
		}
		t.absoluteUnix = dt.Unix()
		return t, true
	}

	// Relative: parse as ical Duration.
	dur, err := triggerProp.Duration()
	if err != nil {
		return t, false
	}
	t.relativeSeconds = int64(dur.Seconds())
	related := strings.ToUpper(strings.TrimSpace(triggerProp.Params.Get(ical.ParamRelated)))
	t.relatedToEnd = related == "END"
	return t, true
}

func computeTriggerUnix(t alarmTemplate, inst EventInstance) int64 {
	if t.absoluteUnix != 0 {
		return t.absoluteUnix
	}
	if t.relatedToEnd {
		return inst.InstanceEndUnix + t.relativeSeconds
	}
	return inst.InstanceStartUnix + t.relativeSeconds
}
