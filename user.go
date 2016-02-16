package main

import "os/user"

var (
	// users is a cache to prevent unnecessary calls to `LookupId`.
	users = map[string]*user.User{}
)

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
