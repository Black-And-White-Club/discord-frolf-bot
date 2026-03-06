package discord

import "testing"

func TestLevelString(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
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
		})
	}
}
