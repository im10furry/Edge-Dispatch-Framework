package edgeagent

import (
	"net/http"
	"testing"
)

func TestParseRangeHeader_Extended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		header     string
		contentLen int64
		wantStart  int64
		wantEnd    int64
		wantErr    bool
	}{
		{
			name:       "valid range",
			header:     "bytes=0-499",
			contentLen: 1000,
			wantStart:  0,
			wantEnd:    499,
		},
		{
			name:       "open end range",
			header:     "bytes=100-",
			contentLen: 200,
			wantStart:  100,
			wantEnd:    199,
		},
		{
			name:       "suffix range",
			header:     "bytes=-50",
			contentLen: 200,
			wantStart:  150,
			wantEnd:    199,
		},
		{
			name:       "suffix larger than content",
			header:     "bytes=-500",
			contentLen: 200,
			wantStart:  0,
			wantEnd:    199,
		},
		{
			name:       "single byte range",
			header:     "bytes=99-99",
			contentLen: 200,
			wantStart:  99,
			wantEnd:    99,
		},
		{
			name:       "end exceeds content length, clamped",
			header:     "bytes=0-500",
			contentLen: 200,
			wantStart:  0,
			wantEnd:    199,
		},
		{
			name:       "end exactly at last byte",
			header:     "bytes=0-199",
			contentLen: 200,
			wantStart:  0,
			wantEnd:    199,
		},
		{
			name:       "empty header",
			header:     "",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "invalid prefix",
			header:     "byte=0-99",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "missing prefix",
			header:     "0-99",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "no dash separator",
			header:     "bytes=100",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "non-numeric start",
			header:     "bytes=abc-99",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "non-numeric end",
			header:     "bytes=0-xyz",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "multiple ranges",
			header:     "bytes=0-10,20-30",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "start equals content length",
			header:     "bytes=200-300",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "start exceeds content length",
			header:     "bytes=300-400",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "negative start",
			header:     "bytes=-1-99",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "start greater than end",
			header:     "bytes=100-50",
			contentLen: 200,
			wantErr:    true,
		},
		{
			name:       "empty suffix value",
			header:     "bytes=-",
			contentLen: 200,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start, end, err := parseRangeHeader(tt.header, tt.contentLen)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if start != tt.wantStart {
				t.Errorf("start = %d, want %d", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end = %d, want %d", end, tt.wantEnd)
			}
		})
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		wantIP     string
	}{
		{
			name:       "remote addr with port",
			remoteAddr: "192.0.2.1:12345",
			wantIP:     "192.0.2.1",
		},
		{
			name:       "remote addr without port",
			remoteAddr: "192.0.2.1",
			wantIP:     "192.0.2.1",
		},
		{
			name:       "x-forwarded-for single",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.5"},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "x-forwarded-for with commas, takes last",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.5, 198.51.100.10"},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "198.51.100.10",
		},
		{
			name:       "x-forwarded-for multiple hops",
			headers:    map[string]string{"X-Forwarded-For": "a, b, c, d"},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "d",
		},
		{
			name:       "x-real-ip",
			headers:    map[string]string{"X-Real-IP": "203.0.113.99"},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "203.0.113.99",
		},
		{
			name:       "x-forwarded-for takes priority over x-real-ip",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.5", "X-Real-IP": "198.51.100.1"},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "empty x-forwarded-for, falls through to x-real-ip",
			headers:    map[string]string{"X-Forwarded-For": "", "X-Real-IP": "198.51.100.1"},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "198.51.100.1",
		},
		{
			name:       "whitespace trimming in x-forwarded-for",
			headers:    map[string]string{"X-Forwarded-For": "  203.0.113.5 , 198.51.100.10  "},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "198.51.100.10",
		},
		{
			name:       "whitespace trimming in x-real-ip",
			headers:    map[string]string{"X-Real-IP": "  203.0.113.5  "},
			remoteAddr: "10.0.0.1:9999",
			wantIP:     "203.0.113.5",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := httptestNewRequest(tt.remoteAddr, tt.headers)
			got := extractClientIP(req)
			if got != tt.wantIP {
				t.Errorf("extractClientIP = %q, want %q", got, tt.wantIP)
			}
		})
	}
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "normal key unchanged",
			key:  "video/stream.m3u8",
			want: "video/stream.m3u8",
		},
		{
			name: "trim spaces",
			key:  "  video/file.mp4  ",
			want: "video/file.mp4",
		},
		{
			name: "backslash to forward slash",
			key:  `video\stream.m3u8`,
			want: "video/stream.m3u8",
		},
		{
			name: "mixed backslash and forward slash",
			key:  `dir\sub/file.txt`,
			want: "dir/sub/file.txt",
		},
		{
			name: "remove dot segment",
			key:  "video/./chunk.ts",
			want: "video/chunk.ts",
		},
		{
			name: "remove double dot segment",
			key:  "a/../b/file.txt",
			want: "a/b/file.txt",
		},
		{
			name: "remove leading dot",
			key:  "./video/file.txt",
			want: "video/file.txt",
		},
		{
			name: "remove leading double dot",
			key:  "../video/file.txt",
			want: "video/file.txt",
		},
		{
			name: "double slashes collapsed",
			key:  "video//file.txt",
			want: "video/file.txt",
		},
		{
			name: "complex mixed invalid segments",
			key:  `./a/b\..\c/./d\\e`,
			want: "a/b/c/d/e",
		},
		{
			name: "empty string",
			key:  "",
			want: "",
		},
		{
			name: "only dots",
			key:  "./../.",
			want: "",
		},
		{
			name: "root path with content",
			key:  "/video/file.txt",
			want: "video/file.txt",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeKey(tt.key)
			if got != tt.want {
				t.Errorf("sanitizeKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func httptestNewRequest(remoteAddr string, headers map[string]string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}
