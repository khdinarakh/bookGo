package data

import (
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Book interface {
		Insert(book *Book, r *http.Request) error
		Get(id int64, r *http.Request) (*Book, error)
		Update(book *Book, r *http.Request) error
		Delete(id int64, r *http.Request) error
		GetAll(title string, content string, genres []string, filters Filters, r *http.Request) ([]*Book, Metadata, error)
	}

	Permissions interface {
		AddForUser(userID int64, codes ...string) error
		GetAllForUser(userID int64) (Permissions, error)
	}

	Tokens interface {
		New(userID int64, ttl time.Duration, scope string) (*Token, error)
		Insert(token *Token) error
		DeleteAllForUser(scope string, userID int64) error
	}

	Users interface {
		Insert(user *User, r *http.Request) error
		GetByEmail(email string, r *http.Request) (*User, error)
		Update(user *User, r *http.Request) error
		GetForToken(tokenScope, tokenPlaintext string) (*User, error)
	}
}

func NewModels(db *pgxpool.Pool) Models {
	return Models{
		Book:        BookModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Users:       UserModel{DB: db},
	}
}
