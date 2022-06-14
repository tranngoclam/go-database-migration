package main_test

import (
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"time"
)

const (
	mysqlDSN       = "root:password@tcp(127.0.0.1:3306)/auth?charset=utf8mb4&parseTime=True&loc=Local"
	migrateSQLUp   = `ALTER TABLE users ADD COLUMN phone_number VARCHAR(127);`
	migrateSQLDown = `ALTER TABLE users DROP COLUMN phone_number;`
)

type User struct {
	ID        uint64    `json:"id" db:"id"`
	FullName  string    `json:"full_name" db:"full_name"`
	Address   string    `json:"address" db:"address"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type (
	gormDB struct {
		client *gorm.DB
	}
	sqlxDB struct {
		client *sqlx.DB
	}
)

func (db *sqlxDB) ListUsers(unsafe bool) ([]*User, error) {
	client := db.client
	if unsafe {
		client = client.Unsafe()
	}
	rows, err := client.Queryx(`SELECT * FROM users;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		var user User
		err := rows.StructScan(&user)
		if err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	return users, nil
}

func (db *sqlxDB) MigrateUp() error {
	_, err := db.client.Exec(migrateSQLUp)
	return err
}

func (db *sqlxDB) MigrateDown() error {
	_, err := db.client.Exec(migrateSQLDown)
	return err
}

func (db *gormDB) ListUsers() ([]*User, error) {
	users := []*User{}
	err := db.client.Debug().Find(&users).Error
	return users, err
}

func (db *gormDB) MigrateUp() error {
	return db.client.Debug().Exec(migrateSQLUp).Error
}

func (db *gormDB) MigrateDown() error {
	return db.client.Debug().Exec(migrateSQLDown).Error
}

func connectGorm() (*gormDB, error) {
	db, err := gorm.Open(mysql.Open(mysqlDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &gormDB{client: db}, err
}

func disconnectGorm(db *gormDB) error {
	sqlDB, err := db.client.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func connectSqlx() (*sqlxDB, error) {
	db, err := sqlx.Connect("mysql", mysqlDSN)
	if err != nil {
		return nil, err
	}

	return &sqlxDB{client: db}, err
}

func disconnectSqlx(db *sqlxDB) error {
	return db.client.Close()
}

func TestMigrationIssue_sqlx_failure(t *testing.T) {
	unsafe := false

	// initial schema with 5 columns
	// +----+-----------+-----------+---------------------+---------------------+
	// | id | full_name | address   | created_at          | updated_at          |
	// +----+-----------+-----------+---------------------+---------------------+
	// |  1 | John Doe  | Singapore | 2022-06-14 11:36:41 | 2022-06-14 11:36:41 |
	// +----+-----------+-----------+---------------------+---------------------+
	db, err := connectSqlx()
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { require.NoError(t, disconnectSqlx(db)) })

	// query unsafely
	users, err := db.ListUsers(unsafe)
	require.NoError(t, err)
	require.Len(t, users, 1)

	user := users[0]
	require.Equal(t, "John Doe", user.FullName)
	require.Equal(t, "Singapore", user.Address)

	// migrate up, add a new column `phone_number` in to `auth.users` table
	// at this moment, some instances of application are still having the old versions,
	// they have to be fully compatible with new schema after the up migration is done,
	// otherwise there will be runtime error happening for the queries
	// +----+-----------+-----------+---------------------+---------------------+---------------|
	// | id | full_name | address   | created_at          | updated_at          | phone_number  |
	// +----+-----------+-----------+---------------------+---------------------+---------------|
	// |  1 | John Doe  | Singapore | 2022-06-14 11:36:41 | 2022-06-14 11:36:41 |               |
	// +----+-----------+-----------+---------------------+---------------------+---------------|
	require.NoError(t, db.MigrateUp())
	t.Cleanup(func() { require.NoError(t, db.MigrateDown()) })

	// select again, simulate real traffic, this returns error due to a strict rule of sqlx
	// by default, unsafe is false so sqlx return error to the application
	// with sqlx, we can solve by setting unsafe to true (calling Unsafe() func)
	// see TestMigrationIssue_sqlx_success for more information
	users, err = db.ListUsers(unsafe)
	require.Error(t, err)
	require.Equal(t, "missing destination name phone_number in *main_test.User", err.Error())
}

func TestMigrationIssue_sqlx_success(t *testing.T) {
	unsafe := true
	db, err := connectSqlx()
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { require.NoError(t, disconnectSqlx(db)) })

	users, err := db.ListUsers(unsafe)
	require.NoError(t, err)
	require.Len(t, users, 1)

	user := users[0]
	require.Equal(t, "John Doe", user.FullName)
	require.Equal(t, "Singapore", user.Address)

	// migrate up
	require.NoError(t, db.MigrateUp())
	t.Cleanup(func() { require.NoError(t, db.MigrateDown()) })

	// select again, simulate real traffic
	users, err = db.ListUsers(unsafe)
	require.NoError(t, err)

	user = users[0]
	require.Equal(t, "John Doe", user.FullName)
	require.Equal(t, "Singapore", user.Address)
}

func TestMigrationIssue_gorm_success(t *testing.T) {
	db, err := connectGorm()
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { require.NoError(t, disconnectGorm(db)) })

	users, err := db.ListUsers()
	require.NoError(t, err)
	require.Len(t, users, 1)

	user := users[0]
	require.Equal(t, "John Doe", user.FullName)
	require.Equal(t, "Singapore", user.Address)

	// migrate up
	require.NoError(t, db.MigrateUp())
	t.Cleanup(func() { require.NoError(t, db.MigrateDown()) })

	// select again, simulate real traffic
	users, err = db.ListUsers()
	require.NoError(t, err)

	user = users[0]
	require.Equal(t, "John Doe", user.FullName)
	require.Equal(t, "Singapore", user.Address)
}
