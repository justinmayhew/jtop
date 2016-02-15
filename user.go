package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type User struct {
	ID   int
	Name string
}

var (
	// users contains all of the users on the system mapped by Uid.
	users = map[int]User{}
)

func init() {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// root:x:0:0:root:/root:/bin/bash
		pieces := strings.Split(line, ":")

		uid, err := strconv.Atoi(pieces[2])
		if err != nil {
			panic(err)
		}

		name := pieces[0]
		users[uid] = User{
			ID:   uid,
			Name: name,
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
