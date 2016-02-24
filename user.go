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

// UserByUID returns a User for a particular UID. An error will be returned
// if a User with that UID does not exist or if the User is not whitelisted.
func UserByUID(uid string) (*user.User, error) {
	if len(UserWhitelist) > 0 {
		// Invoked with the --users argument, ensure `uid` is whitelisted.
		for _, user := range UserWhitelist {
			if user.Uid == uid {
				return userByUID(uid)
			}
		}
		return nil, ErrNotWhitelisted
	}

	return userByUID(uid)
}

func userByUID(uid string) (*user.User, error) {
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
