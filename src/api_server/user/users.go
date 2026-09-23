package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

var InvalidCredentialsError = errors.New("invalid credentials")

type User struct {
	Token string
	UID   uint
	Name  string
}

type UsersManager struct {
	db    *sql.DB
	users []User
}

func CreateUsersManager(db *sql.DB) (*UsersManager, error) {
	var manager UsersManager

	manager.db = db

	_, err := db.Exec(`
create table if not exists users (
    uid serial primary key,
    name text not null unique,
    password_hash text not null
)`)
	if err != nil {
		return nil, err
	}

	return &manager, nil
}

func hashPassword(password string) (string, error) {
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPasswordBytes), nil
}

func createToken() string {
	tokenBuff := make([]byte, 16)
	rand.Read(tokenBuff)
	return hex.EncodeToString(tokenBuff)

}

func (this *UsersManager) RegisterUser(ctx context.Context, name string, password string) (*User, error) {

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	insertedUserRow := this.db.QueryRowContext(ctx, "insert into users (name,password_hash) values ($1, $2) returning uid", name, hashedPassword)
	if err != nil {
		return nil, err
	}
	var uid uint
	err = insertedUserRow.Scan(&uid)
	if err != nil {
		return nil, err
	}

	this.users = append(this.users, User{createToken(), uid, name})

	newUser := &this.users[len(this.users)-1]

	return newUser, nil
}

func (this *UsersManager) LogInUser(ctx context.Context, name string, password string) (*User, error) {

	query := this.db.QueryRowContext(ctx, "select uid, password_hash from users where name=$1", name)

	var uid uint
	var passwordHash string
	err := query.Scan(&uid, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, InvalidCredentialsError
	} else if err != nil {
		return nil, err
	} else if err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, InvalidCredentialsError
	}

	for i := range this.users {
		user := &this.users[i]
		if user.Name == name {
			return user, nil
		}
	}

	this.users = append(this.users, User{createToken(), uid, name})

	return &this.users[len(this.users)-1], nil
}
