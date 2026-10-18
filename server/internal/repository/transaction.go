package repository

import (
	"context"
	"database/sql"
)

type queryConnection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func withTransaction(ctx context.Context, connection queryConnection, operation func(queryConnection) error) error {
	if _, ok := connection.(*sql.Tx); ok {
		return operation(connection)
	}
	transaction, err := connection.(*sql.DB).BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if err := operation(transaction); err != nil {
		return err
	}
	return transaction.Commit()
}

func (repository *SQLiteChoreRepository) Transact(ctx context.Context, operation func(ChoreRepository, ChoreAssignmentRepository, UserRepository, ChoreSeriesRepository) error) error {
	return withTransaction(ctx, repository.database, func(connection queryConnection) error {
		return operation(&SQLiteChoreRepository{connection}, &SQLiteChoreAssignmentRepository{connection}, &SQLiteUserRepository{connection}, &SQLiteChoreSeriesRepository{connection})
	})
}
