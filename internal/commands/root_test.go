package commands

import (
	"errors"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestCommandErrorCodesAreStable(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{sessionv2.ErrStaleFrontier, "stale_state"},
		{sessionv2.ErrConflict, "semantic_conflict"},
		{sessionv2.ErrAmbiguous, "ambiguous_session"},
		{sessionv2.ErrCorruption, "state_corruption"},
		{small.ErrStateBusy, "writer_busy"},
		{errors.New("other"), "invalid_request"},
	}
	for _, test := range cases {
		if got := commandErrorCode(test.err); got != test.want {
			t.Errorf("commandErrorCode(%v) = %q, want %q", test.err, got, test.want)
		}
	}
}
