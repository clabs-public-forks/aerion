package backend

import (
	"reflect"
	"strings"
	"testing"
)

// TestDisabledBridgeReturnsNoErrors checks EXT_RULES R16: with the extension
// disabled, every Calendar_* method is a no-op that returns no error.
func TestDisabledBridgeReturnsNoErrors(t *testing.T) {
	b := reflect.ValueOf(&CalendarBridge{}) // nil SettingsStore → disabled
	errType := reflect.TypeFor[error]()
	for i := range b.NumMethod() {
		m := b.Type().Method(i)
		if !strings.HasPrefix(m.Name, "Calendar_") {
			continue
		}
		fn := b.Method(i)
		args := make([]reflect.Value, fn.Type().NumIn())
		for j := range args {
			args[j] = reflect.Zero(fn.Type().In(j))
		}
		out := fn.Call(args)
		if len(out) == 0 {
			continue
		}
		last := out[len(out)-1]
		if last.Type() == errType && !last.IsNil() {
			t.Errorf("%s returned %v while disabled", m.Name, last.Interface())
		}
	}
}
