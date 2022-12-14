// Package schema contains the database schema, migrations and seeding data.
package schema

import (
	"github.com/dimiro1/darwin"
	"github.com/jmoiron/sqlx"
)

// Migrate attempts to bring the schema for db up to date with the migrations
// defined in this package.
func Migrate(db *sqlx.DB) error {
	driver := darwin.NewGenericDriver(db.DB, darwin.PostgresDialect{})
	d := darwin.New(driver, migrations, nil)
	return d.Migrate()
}

var migrations = []darwin.Migration{
	{
		Version:     1.1,
		Description: "Create table users",
		Script: `
CREATE TABLE users (
    user_id UUID,
    name TEXT,
    email TEXT UNIQUE,
    roles TEXT[],
    password_hash TEXT,
    date_created TIMESTAMP,
    date_updated TIMESTAMP,

    PRIMARY KEY (user_id)
);`,
	},
	{
		Version:     1.2,
		Description: "Create table products",
		Script: `
CREATE TABLE products (
    product_id UUID,
    name TEXT,
    cost INT,
    quantity INT,
    date_created TIMESTAMP,
    date_updated TIMESTAMP,

    PRIMARY KEY (product_id)

);`,
	},
	{
		Version:     1.3,
		Description: "Create table sales",
		Script: `
CREATE TABLE sales (
    sale_id UUID,
    product_id UUID,
    quantity INT,
    paid INT,
    date_created TIMESTAMP,

    PRIMARY KEY (sale_id),
    FOREIGN KEY (product_id) REFERENCES products(product_id) ON DELETE CASCADE
);`,
	},
	{
		Version:     2.1,
		Description: "Alter table products with user column",
		Script: `
ALTER TABLE products
    ADD COLUMN user_id UUID DEFAULT '00000000-0000-0000-0000-000000000000';
`,
	},
}
