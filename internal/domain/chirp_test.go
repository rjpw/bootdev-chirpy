package domain_test

import (
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

func TestChirpFilter(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "clean message",
			body: "I had something interesting for breakfast",
			want: "I had something interesting for breakfast",
		},
		{
			name: "bad word filter 1",
			body: "I hear Mastodon is better than Chirpy. sharbert I need to migrate",
			want: "I hear Mastodon is better than Chirpy. **** I need to migrate",
		},
		{
			name: "bad word filter 2",
			body: "I really need a kerfuffle to go to bed sooner, Fornax !",
			want: "I really need a **** to go to bed sooner, **** !",
		},
		{
			name: "body too long",
			body: "lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
			want: "lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.FilterChirp(tc.body)
			if got != tc.want {
				t.Errorf("Expected %s and got %s", tc.want, got)
			}
		})
	}
}
