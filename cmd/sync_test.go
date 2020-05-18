package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ParseTimeSpec(t *testing.T) {
	cases := []struct {
		startSpec string
		endSpec   string
		start     time.Time
		end       time.Time
	}{
		{
			startSpec: "20200404",
			endSpec:   "20200404",
			start:     time.Date(2020, 04, 04, 00, 00, 00, 00, time.Local),
			end:       time.Date(2020, 04, 04, 23, 59, 00, 00, time.Local),
		},
		{
			startSpec: "20200404",
			endSpec:   "20200405",
			start:     time.Date(2020, 04, 04, 00, 00, 00, 00, time.Local),
			end:       time.Date(2020, 04, 05, 23, 59, 00, 00, time.Local),
		},
	}

	for _, tt := range cases {
		s, e, err := parseTimeSpec(tt.startSpec, tt.endSpec)
		assert.NoError(t, err, "unexpected error %v", err)
		assert.Equal(t, tt.start, s, "expected start %s but got %s", tt.start, s)
		assert.Equal(t, tt.end, e, "expected end %s but got %s", tt.end, e)
	}
}
