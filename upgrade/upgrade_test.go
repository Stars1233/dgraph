/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package upgrade

import (
	"reflect"
	"testing"
)

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version *version
		want    string
	}{
		{
			name:    "nil version",
			version: nil,
			want:    "v0.0.0",
		}, {
			name:    "zero version",
			version: &version{},
			want:    "v0.0.0",
		}, {
			name:    "prior to CalVer versioning",
			version: &version{major: 1, minor: 2, patch: 1},
			want:    "v1.2.1",
		}, {
			name:    "CalVer versioning",
			version: &version{major: 20, minor: 3, patch: 1},
			want:    "v20.03.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	v1 := &version{major: 20, minor: 3, patch: 1}
	v2 := &version{major: 20, minor: 3, patch: 2}
	v3 := &version{major: 20, minor: 7, patch: 0}
	v4 := &version{major: 1, minor: 2, patch: 0}

	tests := []struct {
		name  string
		this  *version
		other *version
		want  versionComparisonResult
	}{
		{
			name:  "v1 == v1, reflexive",
			this:  v1,
			other: v1,
			want:  equal,
		}, {
			name:  "v1 < v2, by patch",
			this:  v1,
			other: v2,
			want:  less,
		}, {
			name:  "v2 > v1, by patch",
			this:  v2,
			other: v1,
			want:  greater,
		}, {
			name:  "v2 < v3, by minor",
			this:  v2,
			other: v3,
			want:  less,
		}, {
			name:  "v1 < v3, transitive, by minor",
			this:  v1,
			other: v3,
			want:  less,
		}, {
			name:  "v1 > v4, by major",
			this:  v1,
			other: v4,
			want:  greater,
		}, {
			name:  "this is nil",
			this:  nil,
			other: v3,
			want:  less,
		}, {
			name:  "other is nil",
			this:  v1,
			other: nil,
			want:  greater,
		}, {
			name:  "both are nil",
			this:  nil,
			other: nil,
			want:  equal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.this.Compare(tt.other); got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVersionFromString(t *testing.T) {
	tests := []struct {
		name       string
		versionStr string
		want       *version
		wantErr    bool
	}{
		{
			name:       "parse mainstream version successfully",
			versionStr: "v1.2.2",
			want:       &version{major: 1, minor: 2, patch: 2},
			wantErr:    false,
		}, {
			name:       "parse beta version successfully",
			versionStr: "v20.03.0-beta.20200320",
			want:       &version{major: 20, minor: 3, patch: 0},
			wantErr:    false,
		}, {
			name:       "error if version doesn't start with v",
			versionStr: "1.2.2",
			want:       nil,
			wantErr:    true,
		}, {
			name:       "error parsing patch version",
			versionStr: "v1.2.2s",
			want:       nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersionFromString(tt.versionStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVersionFromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}
