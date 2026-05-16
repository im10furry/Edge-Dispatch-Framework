package main

import "testing"

func TestParseHTTPRange(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		fileSize  int64
		wantStart int64
		wantEnd   int64
		wantErr   bool
	}{
		{name: "explicit", header: "bytes=0-99", fileSize: 200, wantStart: 0, wantEnd: 99},
		{name: "open end", header: "bytes=100-", fileSize: 200, wantStart: 100, wantEnd: 199},
		{name: "suffix", header: "bytes=-50", fileSize: 200, wantStart: 150, wantEnd: 199},
		{name: "suffix larger than file", header: "bytes=-500", fileSize: 200, wantStart: 0, wantEnd: 199},
		{name: "invalid suffix", header: "bytes=-x", fileSize: 200, wantErr: true},
		{name: "invalid prefix", header: "byte=0-99", fileSize: 200, wantErr: true},
		{name: "start beyond file", header: "bytes=300-400", fileSize: 200, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd, err := parseHTTPRange(tt.header, tt.fileSize)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHTTPRange: %v", err)
			}
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("range = %d-%d, want %d-%d", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
