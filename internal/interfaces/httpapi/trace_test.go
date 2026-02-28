package httpapi

import "testing"

func TestShouldCreateHTTPAPISpan(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "handler span", in: "httpapi.Handler.GetDashboard", want: true},
		{name: "middleware span", in: "httpapi.RequestLogging", want: false},
		{name: "helper span", in: "httpapi.writeError", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldCreateHTTPAPISpan(tt.in)
			if got != tt.want {
				t.Fatalf("shouldCreateHTTPAPISpan(%q)=%v want=%v", tt.in, got, tt.want)
			}
		})
	}
}
