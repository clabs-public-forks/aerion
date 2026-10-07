# Issues

## Open

### `TestSerializeVEVENT_AllDay` fails outside UTC

`extensions/calendar/backend/event_crud_test.go` builds the all-day event at UTC midnight, but the serializer formats the date in local time, so on hosts west of UTC (e.g. PDT) `DTSTART;VALUE=DATE` is the previous day and `make test` fails. Passes with `TZ=UTC`. Decide whether all-day dates should be serialized in UTC or local time, then fix the code or the test.
