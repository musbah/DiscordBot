package main

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4"
)

type user struct {
	userID string
	level  int
}

func lookUpNewUsers(guildMembers []*discordgo.Member, botID string) ([]user, error) {
	newUsers := []user{}

	rows, err := dbPool.Query(context.Background(), "SELECT user_id FROM users")
	if err != nil {
		return newUsers, err
	}

	defer rows.Close()

	currentUsers := make(map[string]bool)
	for rows.Next() {
		userID := ""
		err = rows.Scan(&userID)
		if err != nil {
			return newUsers, err
		}

		currentUsers[userID] = true
	}

	if rows.Err() != nil {
		return newUsers, rows.Err()
	}

	for _, member := range guildMembers {

		_, exists := currentUsers[member.User.ID]

		if !exists && botID != member.User.ID {
			//TODO: make default status a constant (maybe from conf)
			newUsers = append(newUsers, user{member.User.ID, 1})
		}
	}

	return newUsers, nil
}

func addUsersToDB(users []user) error {

	// Instead of doing multiple insert statements, inserting all the users using copyFrom
	_, err := dbPool.CopyFrom(context.Background(), pgx.Identifier{"users"}, []string{"user_id", "level"},
		pgx.CopyFromSlice(len(users), func(i int) ([]interface{}, error) {
			return []interface{}{users[i].userID, users[i].level}, nil
		}))

	return err
}

func getUserStatus(userID string) (user, error) {

	user := user{userID: userID}
	err := dbPool.QueryRow(context.Background(), "SELECT level FROM users WHERE user_id=$1", userID).Scan(&user.level)
	if err != nil {
		return user, err
	}

	return user, nil
}

func doesUserExistInDB(userID string) (bool, error) {
	exists := false
	err := dbPool.QueryRow(context.Background(), "SELECT true FROM users WHERE user_id=$1", userID).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return exists, nil
		}
		return exists, err
	}

	return exists, nil
}

func levelup(userID string) error {
	commandTag, err := dbPool.Exec(context.Background(), "UPDATE users SET level = level + 1 WHERE user_id=$1", userID)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() != 1 {
		return errors.New("No row found to be updated")
	}

	return nil
}
