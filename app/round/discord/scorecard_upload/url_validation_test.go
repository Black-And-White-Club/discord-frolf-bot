package scorecardupload

import "testing"

func Test_validateUDiscURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{
			name:    "valid root domain",
			rawURL:  "https://udisc.com/scorecards/abc.csv",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			rawURL:  "https://app.udisc.com/rounds/123",
			wantErr: false,
		},
		{
			name:    "http is rejected",
			rawURL:  "http://udisc.com/scorecards/abc.csv",
			wantErr: true,
		},
		{
			name:    "non-udisc host is rejected",
			rawURL:  "https://example.com/scorecards/abc.csv",
			wantErr: true,
		},
		{
			name:    "ip host is rejected",
			rawURL:  "https://127.0.0.1/scorecards/abc.csv",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUDiscURL(tt.rawURL)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.rawURL)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.rawURL, err)
			}
		})
	}
}
