package forward

import (
	"context"
	"fmt"
)

type (
	Credentials struct {
		Username  string
		Password  string
		SharedKey string
	}

	AuthClient func(ctx context.Context) (Credentials, error)
	AuthServer func(ctx context.Context, username string) (Credentials, error)
)

func StaticAuthClient(cred Credentials) AuthClient {
	return func(_ context.Context) (Credentials, error) {
		return cred, nil
	}
}

func StaticAuthServer(cred Credentials) AuthServer {
	return func(_ context.Context, username string) (Credentials, error) {
		if username == cred.Username {
			return cred, nil
		}

		return Credentials{}, fmt.Errorf("unknown username '%s'", username)
	}
}
