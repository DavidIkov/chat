package users

import (
	"chat/src/shared"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPasswordBytes), nil
}

func createSessionToken() string {
	tokenBuff := make([]byte, 16)
	rand.Read(tokenBuff)
	return hex.EncodeToString(tokenBuff)

}

func (this *UsersService) RegisterUser(ctx context.Context, name string, password string) (UserSession, error) {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return UserSession{}, err
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	var nameTaken bool
	err = this.db.QueryRowContext(ctx, "select exists (select 1 from users where name = $1)", name).Scan(&nameTaken)
	if err != nil {
		return UserSession{}, err
	}
	if nameTaken {
		return UserSession{}, DuplicateUserNameError
	}

	insertedUserRow := this.db.QueryRowContext(ctx, "insert into users (name,password_hash) values ($1, $2) returning uid", name, hashedPassword)
	var uid uint
	err = insertedUserRow.Scan(&uid)
	if err != nil {
		return UserSession{}, err
	}

	session := UserSession{createSessionToken(), uid}
	this.sessions = append(this.sessions, session)

	return session, nil
}

func (this *UsersService) LogInUser(ctx context.Context, name string, password string) (UserSession, error) {

	query := this.db.QueryRowContext(ctx, "select uid, password_hash from users where name=$1", name)

	var uid uint
	var passwordHash string
	err := query.Scan(&uid, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return UserSession{}, InvalidCredentialsError
	} else if err != nil {
		return UserSession{}, err
	} else if err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return UserSession{}, InvalidCredentialsError
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	for i := range this.sessions {
		if this.sessions[i].UID == uid {
			return this.sessions[i], nil
		}
	}

	session := UserSession{createSessionToken(), uid}
	this.sessions = append(this.sessions, session)

	return session, nil
}

func (this *UsersService) LogOutUser(ctx context.Context, token string) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for i := range this.sessions {
		if this.sessions[i].Token == token {
			this.sessions[i] = this.sessions[len(this.sessions)-1]
			this.sessions = this.sessions[:len(this.sessions)-1]
			return nil
		}
	}
	return TokenNotFoundError
}

func (this *UsersService) DeleteUser(ctx context.Context, uid uint) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if _, err := this.db.ExecContext(ctx, "delete from users where uid = $1", uid); err != nil {
		return err
	}

	for i := 0; i < len(this.sessions); {
		if this.sessions[i].UID == uid {
			this.sessions[i] = this.sessions[len(this.sessions)-1]
			this.sessions = this.sessions[:len(this.sessions)-1]
			continue
		}
		i++
	}
	return nil
}

func (this *UsersService) GetUsers(ctx context.Context, uids []uint) ([]User, error) {
	rows, err := this.db.QueryContext(ctx, "select uid, name from users where uid = any($1)", pq.Array(uids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0, len(uids))
	for rows.Next() {
		var user User
		if err = rows.Scan(&user.UID, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (this *UsersService) GetUserUIDByToken(token string) (shared.UID, error) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for i := range this.sessions {
		if this.sessions[i].Token == token {
			return shared.UID(this.sessions[i].UID), nil
		}
	}
	return 0, TokenNotFoundError
}
