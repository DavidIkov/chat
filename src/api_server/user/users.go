package user

import (
	"chat/src/shared_api"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
)


type User struct {
	Token string
	UID   uint
	Name  string
}


type UsersManager struct {
	db         *sql.DB
	uidCounter uint
	users      []User
}


func CreateUsersManager(db *sql.DB) (*UsersManager, error) {
	var manager UsersManager

	_, err := db.Exec(`
create table if not exists users (
    uid serial primary key,
    name varchar(?) not null,
    password_hash text not null
)`, shared_api.UserNameMaxLen)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(`select max(uid) from users`).Scan(&manager.uidCounter)
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


func (this *UsersManager) RegisterUser(ctx context.Context, name string, password string) (*User, error) {
	tokenBuff := make([]byte, 16)
	rand.Read(tokenBuff)

	this.users = append(this.users, User{hex.EncodeToString(tokenBuff), this.uidCounter, name})
	this.uidCounter++

	newUser := &this.users[len(this.users)-1]

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	_, err = this.db.ExecContext(ctx, "insert into users (name,password_hash) values (?, ?)", name, hashedPassword)
	if err != nil {
		return nil, err
	}

	log.Println("Registered a new user:\n", *newUser)

	return newUser, nil
}


/*func (this *UsersManager) LogInUser(name string, password string) (*User, error) {
    //TODO
}*/
