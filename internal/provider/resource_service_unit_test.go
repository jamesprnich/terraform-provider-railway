package provider

import "testing"

// TestParseServiceImportId exercises the pure parser behind
// ServiceResource.ImportState. The parser is factored out of the framework
// so all edge cases can be verified without the plugin-testing scaffold.
func TestParseServiceImportId(t *testing.T) {
	const (
		validServiceId = "11111111-2222-3333-4444-555555555555"
		validEnvId     = "22222222-3333-4444-5555-666666666666"
	)

	tests := []struct {
		name        string
		raw         string
		strict      bool
		wantSvc     string
		wantEnv     string
		wantErr     bool
		wantSummary string
	}{
		{
			name:    "compound form under strict mode",
			raw:     validServiceId + ":" + validEnvId,
			strict:  true,
			wantSvc: validServiceId,
			wantEnv: validEnvId,
		},
		{
			name:    "compound form under permissive mode",
			raw:     validServiceId + ":" + validEnvId,
			strict:  false,
			wantSvc: validServiceId,
			wantEnv: validEnvId,
		},
		{
			name:    "bare id under permissive mode",
			raw:     validServiceId,
			strict:  false,
			wantSvc: validServiceId,
			wantEnv: "",
		},
		{
			name:        "bare id under strict mode is rejected",
			raw:         validServiceId,
			strict:      true,
			wantErr:     true,
			wantSummary: "environment_id required under strict env-scoping",
		},
		{
			name:        "empty string",
			raw:         "",
			strict:      false,
			wantErr:     true,
			wantSummary: "Unexpected Import Identifier",
		},
		{
			name:        "colon prefix with missing service_id",
			raw:         ":" + validEnvId,
			strict:      false,
			wantErr:     true,
			wantSummary: "Unexpected Import Identifier",
		},
		{
			name:        "trailing colon with empty environment_id",
			raw:         validServiceId + ":",
			strict:      false,
			wantErr:     true,
			wantSummary: "Unexpected Import Identifier",
		},
		{
			name:        "trailing colon with empty environment_id under strict mode is still malformed, not a strict-mode error",
			raw:         validServiceId + ":",
			strict:      true,
			wantErr:     true,
			wantSummary: "Unexpected Import Identifier",
		},
		{
			name:    "extra colon in the environment_id half is preserved verbatim",
			raw:     validServiceId + ":" + validEnvId + ":extra",
			strict:  true,
			wantSvc: validServiceId,
			wantEnv: validEnvId + ":extra",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, env, err := parseServiceImportId(tc.raw, tc.strict)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseServiceImportId(%q, strict=%v) succeeded with (%q, %q); want error with summary %q",
						tc.raw, tc.strict, svc, env, tc.wantSummary)
				}
				if err.summary != tc.wantSummary {
					t.Errorf("parseServiceImportId(%q, strict=%v) returned summary %q; want %q",
						tc.raw, tc.strict, err.summary, tc.wantSummary)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseServiceImportId(%q, strict=%v) unexpected error: %+v",
					tc.raw, tc.strict, err)
			}
			if svc != tc.wantSvc {
				t.Errorf("parseServiceImportId(%q, strict=%v) service = %q; want %q",
					tc.raw, tc.strict, svc, tc.wantSvc)
			}
			if env != tc.wantEnv {
				t.Errorf("parseServiceImportId(%q, strict=%v) environment_id = %q; want %q",
					tc.raw, tc.strict, env, tc.wantEnv)
			}
		})
	}
}
