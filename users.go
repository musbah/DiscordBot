package main

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4"
)

const (
	level        = 1
	exp          = 0
	maxHP        = 100
	maxMP        = 50
	strength     = 1
	agility      = 1
	intelligence = 1
	defence      = 1
	mDefence     = 1
)

type user struct {
	userID       string
	level        int
	exp          int
	maxHp        int
	currentHP    int
	maxMp        int
	currentMP    int
	strength     int
	agility      int
	intelligence int
	defence      int
	mDefence     int
}

func (user user) String() string {
	userString := "Level " + strconv.Itoa(user.level) + "\n" +
		"Exp " + strconv.Itoa(user.exp) + "\n" +
		"Max HP " + strconv.Itoa(user.maxHp) +
		"\tCurrent HP " + strconv.Itoa(user.currentHP) + "\n" +
		"Max MP " + strconv.Itoa(user.maxMp) +
		"\tCurrent MP " + strconv.Itoa(user.currentMP) + "\n" +
		"Strength " + strconv.Itoa(user.strength) + "\n" +
		"Agility " + strconv.Itoa(user.agility) + "\n" +
		"Intelligence " + strconv.Itoa(user.intelligence) + "\n" +
		"Defence " + strconv.Itoa(user.defence) + "\n" +
		"Magic Defence " + strconv.Itoa(user.mDefence)

	return userString
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
		var userID uint64
		err = rows.Scan(&userID)
		if err != nil {
			return newUsers, err
		}

		currentUsers[intToString(userID)] = true
	}

	if rows.Err() != nil {
		return newUsers, rows.Err()
	}

	for _, member := range guildMembers {

		_, exists := currentUsers[member.User.ID]

		if !exists && botID != member.User.ID {
			newUsers = append(newUsers, createDefaultUserStruct(member.User.ID))
		}
	}

	return newUsers, nil
}

func addUsersToDB(users []user) error {

	// Instead of doing multiple insert statements, inserting all the users using copyFrom
	_, err := dbPool.CopyFrom(context.Background(), pgx.Identifier{"users"},
		[]string{"user_id", "level", "exp", "max_hp", "current_hp",
			"max_mp", "current_mp", "strength", "agility", "intelligence", "defence", "magic_defence"},
		pgx.CopyFromSlice(len(users), func(i int) ([]interface{}, error) {
			user := users[i]
			return []interface{}{stringToInt(user.userID), user.level, user.exp, user.maxHp, user.currentHP,
				user.maxMp, user.currentMP, user.strength, user.agility, user.intelligence, user.defence, user.mDefence}, nil
		}))

	return err
}

func getUserStatus(userID string) (user, error) {

	user := user{userID: userID}
	err := dbPool.QueryRow(context.Background(),
		"SELECT level,exp,max_hp,current_hp,max_mp,current_mp,strength,agility,"+
			"intelligence,defence,magic_defence FROM users WHERE user_id=$1",
		userID).Scan(&user.level, &user.exp, &user.maxHp, &user.currentHP, &user.maxMp, &user.currentMP,
		&user.strength, &user.agility, &user.intelligence, &user.defence, &user.mDefence)
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

func createDefaultUserStruct(userID string) user {
	return user{
		userID:       userID,
		level:        level,
		exp:          exp,
		maxHp:        maxHP,
		currentHP:    maxHP,
		maxMp:        maxMP,
		currentMP:    maxMP,
		strength:     strength,
		agility:      agility,
		intelligence: intelligence,
		defence:      defence,
		mDefence:     mDefence,
	}
}

// Returns exp required for the next level
func nextLevelExp(level int) int {
	return int(math.Round((4 * (math.Pow(float64(level), 3))) / 5))
}
