package backend

import (
	"strings"

	"github.com/emersion/go-ical"
)

// composerProps are the master VEVENT properties the composer owns. An edit
// replaces these and leaves every other property (X-props, CATEGORIES,
// URL, SEQUENCE, ...) as the server or another client wrote it.
var composerProps = []string{
	ical.PropDateTimeStamp, ical.PropSummary, ical.PropDescription, icsPropAltDesc,
	ical.PropLocation, icsPropTransp, icsPropClass, ical.PropDateTimeStart,
	ical.PropDateTimeEnd, ical.PropDuration, ical.PropRecurrenceRule,
	ical.PropOrganizer, ical.PropAttendee,
}

// mergeRRule returns the RRULE for a whole-series edit. The composer only
// shows FREQ, UNTIL and COUNT, so the other parts of the existing rule
// (INTERVAL, BYDAY, WKST, ...) are kept while the frequency is unchanged.
// spec.Keep returns the existing rule as is. allDay writes UNTIL as a DATE.
func mergeRRule(old string, spec *RecurrenceSpec, allDay bool) string {
	if spec == nil {
		return ""
	}
	old = strings.TrimPrefix(old, "RRULE:")
	var kept []string
	sameFreq := false
	for _, part := range strings.Split(old, ";") {
		name, value, _ := strings.Cut(part, "=")
		switch strings.ToUpper(name) {
		case "":
		case "FREQ":
			sameFreq = strings.EqualFold(value, spec.Freq)
		case "UNTIL", "COUNT":
		default:
			kept = append(kept, part)
		}
	}
	if !sameFreq {
		return rruleText(spec, allDay)
	}
	if spec.Keep {
		return old
	}
	fresh := strings.Split(rruleText(spec, allDay), ";")
	return strings.Join(append(append(fresh[:1:1], kept...), fresh[1:]...), ";")
}

// primaryReminderMinutes returns the offset in minutes of the first VALARM
// the composer can show (a trigger at or before the start), or nil when
// there is none.
func primaryReminderMinutes(blob string) *int {
	tmpls, err := parseAlarmTemplates(blob)
	if err != nil {
		return nil
	}
	for _, t := range tmpls {
		if t.absoluteUnix != 0 || t.relatedToEnd || t.relativeSeconds > 0 {
			continue
		}
		m := int(-t.relativeSeconds / 60)
		return &m
	}
	return nil
}

// sameReminder reports whether the composer's reminder matches the one it
// was shown for the existing event.
func sameReminder(existing *int, in *ReminderSpec) bool {
	if existing == nil || in == nil {
		return existing == nil && in == nil
	}
	return *existing == in.OffsetMinutes
}

// mergeVEVENT applies a whole-series edit to master's stored VCALENDAR.
// Only composerProps are replaced. Existing VALARMs stay unless the reminder
// changed. EXDATEs, RDATEs and override VEVENTs stay while the series timing
// (start, all-day, zone, RRULE) is unchanged, since their RECURRENCE-IDs
// still match; keepExceptions reports that. A blob that can't be decoded is
// replaced by a fresh one.
func mergeVEVENT(master Event, in EventInput) (blob string, keepExceptions bool, err error) {
	rrule := rruleText(in.Recurrence, in.IsAllDay)
	fresh := newVEVENT(master.UID, in, rrule)
	cal, derr := ical.NewDecoder(strings.NewReader(master.ICSBlob)).Decode()
	masterIdx := -1
	if derr == nil {
		masterIdx, _ = classifyVEvents(cal)
	}
	if masterIdx < 0 {
		blob, err = serializeVEVENT(master.UID, in)
		return blob, false, err
	}

	keepExceptions = in.DTStartUnix == master.DTStartUnix && in.IsAllDay == master.IsAllDay &&
		in.TZName == master.TZName && rrule == strings.TrimPrefix(master.RRuleText, "RRULE:")

	vevent := cal.Children[masterIdx]
	for _, name := range composerProps {
		vevent.Props.Del(name)
	}
	for name, props := range fresh.Props {
		vevent.Props[name] = props
	}
	if !sameReminder(primaryReminderMinutes(master.ICSBlob), in.Reminder) {
		children := vevent.Children[:0]
		for _, c := range vevent.Children {
			if c.Name != ical.CompAlarm {
				children = append(children, c)
			}
		}
		vevent.Children = append(children, fresh.Children...)
	}
	if !keepExceptions {
		vevent.Props.Del(ical.PropExceptionDates)
		vevent.Props.Del(ical.PropRecurrenceDates)
		children := cal.Children[:0]
		for _, c := range cal.Children {
			if c.Name != ical.CompEvent || c.Props.Get(ical.PropRecurrenceID) == nil {
				children = append(children, c)
			}
		}
		cal.Children = children
	}
	blob, err = encodeICS(cal)
	return blob, keepExceptions, err
}
