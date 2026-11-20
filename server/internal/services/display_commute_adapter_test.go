package services

import (
	"context"
	"errors"
	"testing"
)

type displayTimezoneSettingsFake struct {
	value string
	err   error
}

func (fake displayTimezoneSettingsFake) Get(context.Context, string) (string, error) {
	return fake.value, fake.err
}
func (displayTimezoneSettingsFake) Set(context.Context, string, string) error { return nil }
func TestDisplayCommuteTimezoneAdapter(t *testing.T) {
	failure := errors.New("read failed")
	for _, test := range []struct {
		name, value, want string
		readErr           error
		decodeErr         bool
	}{{name: "timezone", value: `{"timezone":"Europe/Paris"}`, want: "Europe/Paris"}, {name: "invalid json", value: `{`, decodeErr: true}, {name: "read error", readErr: failure}} {
		t.Run(test.name, func(t *testing.T) {
			got := NewDisplayCommuteTimezone(displayTimezoneSettingsFake{test.value, test.readErr}).Timezone(context.Background())
			if got.Value != test.want || !errors.Is(got.ReadError, test.readErr) || (got.DecodeError != nil) != test.decodeErr {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
