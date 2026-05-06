package auth_test

import (
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/auth"
)

func TestPasswordHashing(t *testing.T) {
	cases := []struct {
		name         string
		password     string
		hash         string
		generateHash bool
		wantMatch    bool
	}{
		{
			name:      "should fail blank hash",
			password:  "password",
			hash:      "",
			wantMatch: false,
		},
		{
			name:         "check generated hash",
			password:     "password",
			generateHash: true,
			wantMatch:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			if tc.generateHash {
				hash, err := auth.HashPassword(tc.password)
				if err != nil {
					t.Errorf("Unexpected error %v", err)
				}
				tc.hash = hash
			}

			gotMatch, _ := auth.CheckPasswordHash(tc.password, tc.hash)
			if gotMatch != tc.wantMatch {
				t.Errorf("Expected %s to get %v and got %v", tc.name, tc.wantMatch, gotMatch)
			}
		})
	}

}
