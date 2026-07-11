package mediavalidation

import (
	"errors"
	"testing"
)

// TestDetectImageContentType is the contract both Application icon (ADR-073) and
// Tenant branding asset (ADR-096) upload paths rely on: fixed magic-byte
// detection, a hard size ceiling, and rejection of anything else (including SVG).
func TestDetectImageContentType(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 4)...)
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0}
	webp := append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WEBP")...)
	gif87 := []byte("GIF87a")
	gif89 := []byte("GIF89a")

	cases := []struct {
		name        string
		data        []byte
		maxBytes    int
		wantType    string
		wantErr     error
		wantErrOnly bool
	}{
		{name: "png", data: png, maxBytes: 1024, wantType: "image/png"},
		{name: "jpeg", data: jpeg, maxBytes: 1024, wantType: "image/jpeg"},
		{name: "webp", data: webp, maxBytes: 1024, wantType: "image/webp"},
		{name: "gif87a", data: gif87, maxBytes: 1024, wantType: "image/gif"},
		{name: "gif89a", data: gif89, maxBytes: 1024, wantType: "image/gif"},
		{name: "empty", data: nil, maxBytes: 1024, wantErr: ErrImageRequired},
		{name: "oversize", data: png, maxBytes: len(png) - 1, wantErr: ErrImageTooLarge},
		{name: "svg rejected", data: []byte("<svg onload=alert(1)></svg>"), maxBytes: 1024, wantErr: ErrImageFormat},
		{name: "plain text rejected", data: []byte("plain text data"), maxBytes: 1024, wantErr: ErrImageFormat},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contentType, err := DetectImageContentType(tc.data, tc.maxBytes)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got err %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if contentType != tc.wantType {
				t.Fatalf("got content type %q, want %q", contentType, tc.wantType)
			}
		})
	}
}
