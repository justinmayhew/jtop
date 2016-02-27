package main

import (
	"errors"
	"os/user"
)

var (
	// UserWhitelist contains the users whitelisted via the --users option.
	UserWhitelist     []*user.User
	ErrNotWhitelisted = errors.New("not monitoring that users processes")

	// users is a cache to prevent unnecessary calls to `LookupId`.
	users = map[string]*user.User{}
)

// UserByUid returns a User for a particular Uid. An error will be returned
// if a User with that Uid does not exist or if the User is not whitelisted.
func UserByUid(uid string) (*user.User, error) {
	if len(UserWhitelist) == 0 {
		return userByUid(uid)
	}
	for _, user := range UserWhitelist {
		if user.Uid == uid {
			return userByUid(uid)
		}
	}
	return nil, ErrNotWhitelisted
}

func userByUid(uid string) (*user.User, error) {
	if user, ok := users[uid]; ok {
		return user, nil
	}

	user, err := user.LookupId(uid)
	if err != nil {
		return nil, err
	}

	users[uid] = user
	return user, nil
}
