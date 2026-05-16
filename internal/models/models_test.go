package models

import "testing"

func TestEndpointURLFormatsIPv6Host(t *testing.T) {
	got := (Endpoint{Scheme: "http", Host: "2001:db8::1", Port: 9090}).URL()
	want := "http://[2001:db8::1]:9090"
	if got != want {
		t.Fatalf("URL() = %q, want %q", got, want)
	}
}
