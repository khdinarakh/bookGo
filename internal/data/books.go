package data

import (
	"books.reading.kz/internal/validator"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

type Book struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Year      int32     `json:"year,omitempty"`
	Pages     Pages     `json:"pages,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   string    `json:"version"`
}

func ValidateBook(v *validator.Validator, book *Book) {
	v.Check(book.Title != "", "title", "must be provided")
	v.Check(len(book.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(len(book.Content) >= 10, "content", "must be more than 10 bytes long")
	v.Check(book.Year != 0, "year", "must be provided")
	v.Check(book.Year >= 1888, "year", "must be greater than 1888")
	v.Check(book.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(book.Pages != 0, "pages", "must be provided")
	v.Check(book.Pages > 0, "pages", "must be a positive integer")
	v.Check(book.Genres != nil, "genres", "must be provided")
	v.Check(len(book.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(book.Genres) <= 5, "genres", "must not contain more than 5 genres")
	// Note that we're using the Unique helper in the line below to check that all
	// values in the input.Genres slice are unique.
	v.Check(validator.Unique(book.Genres), "genres", "must not contain duplicate values")
	// Use the Valid() method to see if any of the checks failed. If they did, then use
	// the failedValidationResponse() helper to send a response to the client, passing
	// in the v.Errors map.
}

type BookModel struct {
	DB *pgxpool.Pool
}

func (b BookModel) Insert(book *Book, r *http.Request) error {
	query := `
		INSERT INTO books (title, year, content, pages, genres)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version`

	args := []any{book.Title, book.Year, book.Content, book.Pages, book.Genres}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	return b.DB.QueryRow(ctx, query, args...).Scan(&book.ID, &book.CreatedAt, &book.Version)
}

func (b BookModel) Get(id int64, r *http.Request) (*Book, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
        SELECT id, created_at, title, content, year, pages, genres, version
        FROM books
        WHERE id = $1`

	var book Book

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := b.DB.QueryRow(ctx, query, id).Scan(
		&book.ID,
		&book.CreatedAt,
		&book.Title,
		&book.Content,
		&book.Year,
		&book.Pages,
		&book.Genres,
		&book.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &book, nil
}

func (b BookModel) Update(book *Book, r *http.Request) error {
	query := `
       UPDATE books
       SET title = $1, content = $2, year = $3, pages = $4, genres = $5, version = uuid_generate_v4()
       WHERE id = $6 AND version = $7
       RETURNING version`

	args := []any{
		book.Title,
		book.Content,
		book.Year,
		book.Pages,
		book.Genres,
		book.ID,
		book.Version,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := b.DB.QueryRow(ctx, query, args...).Scan(&book.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil

}

func (b BookModel) Delete(id int64, r *http.Request) error {

	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM books
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	result, err := b.DB.Exec(ctx, query, id)

	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (b BookModel) GetAll(title string, content string, genres []string, filters Filters, r *http.Request) ([]*Book, Metadata, error) {
	//  to_tsvector('simple', title) function takes a movie title and splits it into lexemes

	//plainto_tsquery('simple', $1) function takes a search value and turns it into a
	//formatted query term that PostgreSQ
	query := fmt.Sprintf(`
		SELECT  count(*) OVER(), id, created_at, title, content, year, pages, genres, version
		FROM books
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	args := []any{title, genres, filters.limit(), filters.offset()}
	rows, err := b.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	books := []*Book{}

	for rows.Next() {
		var book Book

		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.CreatedAt,
			&book.Title,
			&book.Content,
			&book.Year,
			&book.Pages,
			&book.Genres,
			&book.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return books, metadata, nil

}
