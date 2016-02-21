package main

import "os/user"

var (
	// users is a cache to prevent unnecessary calls to `LookupId`.
	users = map[string]*user.User{}
)

// UserByUID returns a User for a particular UID. An error will be returned
// if a User with that UID does not exist.
func UserByUID(uid string) (*user.User, error) {
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
