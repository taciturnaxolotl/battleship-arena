package storage

import (
	"database/sql"
	"strings"
	"time"
)

type User struct {
	ID          int
	Username    string
	Name        string
	Bio         string
	Link        string
	PublicKey   string
	CreatedAt   time.Time
	LastLoginAt time.Time
}

func GetUserByUsername(username string) (*User, error) {
	var u User
	var lastLogin sql.NullTime
	err := DB.QueryRow(
		`SELECT id, username, name, bio, link, public_key, created_at, last_login_at 
		 FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Name, &u.Bio, &u.Link, &u.PublicKey, &u.CreatedAt, &lastLogin)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	if lastLogin.Valid {
		u.LastLoginAt = lastLogin.Time
	}
	
	return &u, nil
}

func GetUserByPublicKey(publicKey string) (*User, error) {
	publicKey = strings.TrimSpace(publicKey)
	
	var u User
	var lastLogin sql.NullTime
	err := DB.QueryRow(
		`SELECT id, username, name, bio, link, public_key, created_at, last_login_at 
		 FROM users WHERE TRIM(public_key) = ?`,
		publicKey,
	).Scan(&u.ID, &u.Username, &u.Name, &u.Bio, &u.Link, &u.PublicKey, &u.CreatedAt, &lastLogin)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	if lastLogin.Valid {
		u.LastLoginAt = lastLogin.Time
	}
	
	return &u, nil
}

func CreateUser(username, name, bio, link, publicKey string) (*User, error) {
	result, err := DB.Exec(
		`INSERT INTO users (username, name, bio, link, public_key, created_at, last_login_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		username, name, bio, link, publicKey, time.Now(), time.Now(),
	)
	if err != nil {
		return nil, err
	}
	
	id, _ := result.LastInsertId()
	return &User{
		ID:          int(id),
		Username:    username,
		Name:        name,
		Bio:         bio,
		Link:        link,
		PublicKey:   publicKey,
		CreatedAt:   time.Now(),
		LastLoginAt: time.Now(),
	}, nil
}

func UpdateUserLastLogin(username string) error {
	_, err := DB.Exec(
		"UPDATE users SET last_login_at = ? WHERE username = ?",
		time.Now(), username,
	)
	return err
}

func UpdateUserProfile(username, name, bio, link string) error {
	_, err := DB.Exec(
		"UPDATE users SET name = ?, bio = ?, link = ? WHERE username = ?",
		name, bio, link, username,
	)
	return err
}

func GetAllUsers() ([]User, error) {
	rows, err := DB.Query(
		`SELECT id, username, name, bio, link, public_key, created_at, last_login_at 
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var u User
		var lastLogin sql.NullTime
		err := rows.Scan(&u.ID, &u.Username, &u.Name, &u.Bio, &u.Link, &u.PublicKey, &u.CreatedAt, &lastLogin)
		if err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			u.LastLoginAt = lastLogin.Time
		}
		users = append(users, u)
	}
	
	return users, rows.Err()
}
