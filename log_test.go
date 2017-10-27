package logrotating

import (
	"testing"
)

func Test_log(t *testing.T) {
	SetLogLevel(LOG_LEVEL_ERROR)
	Error("test")
}
