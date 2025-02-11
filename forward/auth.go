package forward

import (
	"context"
	"fmt"
)

type (
	AuthClient func(ctx context.Context) (username, password string)
	AuthServer func(ctx context.Context, username string) (password string, err error)
)

func StaticAuthClient(username, password string) AuthClient {
	return func(_ context.Context) (username string, password string) {
		return username, password
	}
}

func StaticAuthServer(username, password string) AuthServer {
	return func(_ context.Context, user string) (string, error) {
		if user == username {
			return password, nil
		}

		return "", fmt.Errorf("username '%s' no found", user)
	}
}
