package utils

import (
	"testing"
	"time"
)

func TestTimeToUnixMilli(t *testing.T) {
	if TimeToUnixMilli(nil) != nil {
		t.Fatal("absent timestamp must stay absent")
	}
	now := time.UnixMilli(1728000000123)
	value := TimeToUnixMilli(&now)
	if value == nil || *value != 1728000000123 {
		t.Fatal(value)
	}
}
