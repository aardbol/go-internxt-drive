package internxtclient_test

import (
	"strings"
	"testing"
	"time"

	client "github.com/StarHack/go-internxt-drive/internxtclient"
)

func TestAuthIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// using its own client (could possibly run in parallel)
	var authClient *client.Client
	var loginResponse *client.LoginResponse

	t.Run("Login", func(t *testing.T) {
		authClient, loginResponse = login(t)
	})

	t.Run("AccessLogin", func(t *testing.T) {
		accessLogin(t, authClient, loginResponse)
	})

	t.Run("AreCredentialsCorrect", func(t *testing.T) {
		areCredentialsCorrect(t, authClient)
	})

	t.Run("Logout", func(t *testing.T) {
		logout(t, authClient)
	})
}

func logout(t *testing.T, authClient *client.Client) {
	// logout
	err := authClient.Auth.Logout(testCtx)
	if err != nil {
		t.Fatalf("error logging out: %v", err)
	}

	// Token revocation can lag the logout response, so poll briefly until
	// the server rejects the credentials instead of sampling once.
	deadline := time.Now().Add(5 * time.Second)
	var correct bool
	for {
		correct, err = authClient.Auth.AreCredentialsCorrect(testCtx, authClient.Config.PasswordHash)
		if err != nil || !correct || time.Now().After(deadline) {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	// The result must either be an Unauthorized error or correct == false.
	if err != nil {
		if !strings.Contains(err.Error(), "Unauthorized") {
			t.Fatalf("unexpected error checking credentials after logout: %v", err)
		}
		return
	}
	if correct {
		t.Fatalf("credentials are still correct after logout")
	}
}

func areCredentialsCorrect(t *testing.T, authClient *client.Client) bool {
	// check if the credentials are correct
	correct, err := authClient.Auth.AreCredentialsCorrect(testCtx, authClient.Config.PasswordHash)
	if err != nil {
		t.Fatalf("error checking credentials: %v", err)
	}
	if !correct {
		t.Fatalf("credentials are not correct")
	}

	return correct
}

func login(t *testing.T) (*client.Client, *client.LoginResponse) {
	authClient := client.NewWithDefaults()
	loginResponse, err := authClient.Auth.Login(testCtx, testEmail)
	if err != nil {
		t.Fatalf("couldn't log in: %v", err)
	}
	if loginResponse == nil {
		t.Fatalf("loginResponse was nil")
	}
	if loginResponse.SKey == "" {
		t.Fatalf("did not receive Skey")
	}

	return authClient, loginResponse
}

func accessLogin(t *testing.T, authClient *client.Client, loginResponse *client.LoginResponse) {
	authClient.UserData.AccessData.User = &client.User{Email: testEmail}
	accessResponse, err := authClient.Auth.AccessLogin(testCtx, loginResponse, testPassword)
	if err != nil {
		t.Fatalf("couldn't log in: %v", err)
	}
	if accessResponse == nil {
		t.Fatalf("accessResponse was nil")
	}
	if accessResponse.NewToken == "" {
		t.Fatalf("did not receive NewToken")
	}
}
