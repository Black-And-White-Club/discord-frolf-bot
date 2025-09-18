package discord

import "testing"

func TestLevelString(t *testing.T) {
	cases := []struct {
		in   Level
		want string
	}{
		{Player, "Player"},
		{Editor, "Editor"},
		{Admin, "Admin"},
		{NoPermission, "Unknown"},
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.want {
			t.Errorf("Level(%v).String()=%q want %q", c.in, got, c.want)
		}
	}
}
