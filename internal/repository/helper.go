package repository

import (
	"iter"

	"github.com/jackc/pgx/v5"
)

func scanRows[T any](rows pgx.Rows, scanFn func(pgx.Rows) (T, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		defer rows.Close()

		for rows.Next() {
			val, err := scanFn(rows)
			if !yield(val, err) || err != nil {
				return
			}
		}

		if err := rows.Err(); err != nil {
			var zero T
			yield(zero, err)
		}
	}
}
