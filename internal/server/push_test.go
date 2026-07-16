package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestShouldPush(t *testing.T) {
	on := boolPtr(true)
	off := boolPtr(false)
	connected := &PlayerState{PublicPlayer: types.PublicPlayer{Connected: true}}
	disconnected := &PlayerState{PublicPlayer: types.PublicPlayer{Connected: false}}

	cases := []struct {
		name   string
		player *PlayerState
		pref   *bool
		want   bool
	}{
		{"nil player", nil, on, false},
		{"connected, pref on", connected, on, false},
		{"disconnected, pref on", disconnected, on, true},
		{"disconnected, pref off", disconnected, off, false},
		{"disconnected, pref nil", disconnected, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldPush(c.player, c.pref); got != c.want {
				t.Errorf("shouldPush() = %v, want %v", got, c.want)
			}
		})
	}
}
