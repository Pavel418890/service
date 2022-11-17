package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pavel418890/service/business/auth"
)

// Success and failure markers.
const (
	success = "\u2713"
	failed  = "\u2717"
)

func TestAuth(t *testing.T) {
	t.Log("Given the need to be able to authenticate and authorizate access.")
	{
		testID := 0
		t.Logf("\tTest %d:\tWhen handling a single user.", testID)
		{
			// Generate a new private key
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}

			// The key id we are staring represents the public key in the
			// public key store.
			const keyID = "fb74bc1d2b4e47a967e2d3728f1d952d9a79456a1005c2df0c4cb9eed6fa49c4"
			lookup := func(kid string) (*rsa.PublicKey, error) {
				switch kid {
				case keyID:
					return &privateKey.PublicKey, nil
				}
				return nil, fmt.Errorf("no public key found for the specified kid: %s", kid)
			}

			a, err := auth.New("RS256", lookup, auth.Keys{keyID: privateKey})
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to create an authenticator: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to create an autnenticator.", success, testID)

			claims := auth.Claims{
				StandardClaims: jwt.StandardClaims{
					Issuer:    "service project",
					Subject:   "user1",
					Audience:  "students",
					ExpiresAt: time.Now().Add(8760 * time.Hour).Unix(),
					IssuedAt:  time.Now().Unix(),
				},
				Roles: []string{auth.RoleAdmin},
			}

			token, err := a.GenerateToken(keyID, claims)
			if err != nil {

				t.Fatalf("\t%s\tTest %d:\tShould be able to generate a JWT: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to generate a JWT.", success, testID)

			parsedClaims, err := a.ValidateToken(token)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to parse the claims: %v", failed, testID, err)
			}

			t.Logf("\t%s\tTest %d:\tShould be able to parse the claims.", success, testID)

			if exp, got := len(claims.Roles), len(parsedClaims.Roles); exp != got {
				t.Logf("\t\tTest %d:\texp: %d", testID, exp)
				t.Logf("\t\tTest %d:\texp: %d", testID, got)
				t.Fatalf("\t%s\tTest %d:\tShould have the expected number of roles: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have the expected number of roles.", success, testID)

			if exp, got := claims.Roles[0], parsedClaims.Roles[0]; exp != got {
				t.Logf("\t\tTest %d:\texp: %s", testID, exp)
				t.Logf("\t\tTest %d:\texp: %s", testID, got)
				t.Fatalf("\t%s\tTest %d:\tShould have the expected number of roles: %v", failed, testID, err)
			}

			t.Logf("\t%s\tTest %d:\tShould have the expected number of roles.", success, testID)
		}
	}
}
