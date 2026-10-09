package backend

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-ical"
)

// tzCache memoizes loadTZ by TZID: time.LoadLocation reads zoneinfo from
// disk on every call, and every blob decode resolves its TZIDs.
var tzCache sync.Map // string -> tzResult

type tzResult struct {
	loc *time.Location
	err error
}

// loadTZ resolves an iCalendar TZID to a location. Besides IANA names it
// accepts Windows zone names (Outlook, Exchange) and IANA names behind a
// vendor prefix such as "/mozilla.org/20050126_1/Europe/Berlin".
func loadTZ(tzid string) (*time.Location, error) {
	if r, ok := tzCache.Load(tzid); ok {
		return r.(tzResult).loc, r.(tzResult).err
	}
	loc, err := resolveTZ(tzid)
	tzCache.Store(tzid, tzResult{loc, err})
	return loc, err
}

func resolveTZ(tzid string) (*time.Location, error) {
	name := strings.Trim(strings.TrimSpace(tzid), `"`)
	if name == "" {
		return nil, fmt.Errorf("empty TZID")
	}
	if l, err := time.LoadLocation(name); err == nil {
		return l, nil
	}
	if iana, ok := windowsZones[name]; ok {
		return time.LoadLocation(iana)
	}
	for i := strings.IndexByte(name, '/'); i >= 0; i = strings.IndexByte(name, '/') {
		name = name[i+1:]
		if name == "" {
			break
		}
		if l, err := time.LoadLocation(name); err == nil {
			return l, nil
		}
	}
	return nil, fmt.Errorf("unknown TZID %q", tzid)
}

// ianaToWindowsZone is the reverse of windowsZones. When several Windows
// names map to one IANA zone, the lexically smallest wins so the choice is
// stable.
var ianaToWindowsZone = func() map[string]string {
	out := make(map[string]string, len(windowsZones))
	for win, iana := range windowsZones {
		if cur, ok := out[iana]; !ok || win < cur {
			out[iana] = win
		}
	}
	return out
}()

// windowsZoneName returns the Windows zone name for an IANA location, or ""
// when no Windows zone maps to it.
func windowsZoneName(loc *time.Location) string {
	return ianaToWindowsZone[loc.String()]
}

// decodeICS decodes the first VCALENDAR in blob and rewrites its TZID
// parameters to IANA names (see normalizeTZIDs). Use it for read-only
// parsing; blobs that are edited and sent back keep their original TZIDs so
// they still match their VTIMEZONE.
func decodeICS(blob string) (*ical.Calendar, error) {
	cal, err := ical.NewDecoder(strings.NewReader(blob)).Decode()
	if err != nil {
		return nil, err
	}
	normalizeTZIDs(cal.Component)
	return cal, nil
}

// normalizeTZIDs rewrites each TZID parameter that go-ical can't load into
// the IANA name loadTZ resolves it to. An unresolvable TZID is removed, so the
// time reads as floating instead of failing to parse.
func normalizeTZIDs(c *ical.Component) {
	for _, props := range c.Props {
		for i := range props {
			tzid := props[i].Params.Get(ical.ParamTimezoneID)
			if tzid == "" {
				continue
			}
			l, err := loadTZ(tzid)
			switch {
			case err != nil:
				props[i].Params.Del(ical.ParamTimezoneID)
			case l.String() != tzid:
				props[i].Params.Set(ical.ParamTimezoneID, l.String())
			}
		}
	}
	for _, child := range c.Children {
		normalizeTZIDs(child)
	}
}
