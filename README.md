# Migration for Golang
Currently only works with `github.com/jmoiron/sqlx`.

Greatly inspired by Phinx ❤️

## How to use
Install : `go get -u github.com/raph6/migration`

Create a `migrations` folder in the root of the project and upload your `.sql` files as `ID_name_of_migration.sql`.

Example: `10001_create_accounts_table.sql`

Can contain several SQL queries such as :
```sql
-- 10001_create_accounts_table.sql
CREATE TABLE `accounts` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `username` VARCHAR(255) NOT NULL,
    `password` VARCHAR(255) NOT NULL
);
```

```sql
-- 10002_add_admin_to_accounts_table.sql
ALTER TABLE `accounts` ADD COLUMN `secret_token` varchar(200) DEFAULT NULL;

ALTER TABLE `accounts` ADD COLUMN `admin` TINYINT(1) NOT NULL DEFAULT 0;
```

Please note, do not set file IDs to 1, 2, 3... 10, 11, 12, otherwise files 10, 11, 12 will be read before 1, 2, 3.

I advise you to start at 10000 and then increment or take the timestamp of the file creation.

## Run
```go
import "github.com/raph6/migration"

func main() {
    var db *sqlx.DB
    db = ...

    migration.Migrate(db)
}
```

## Incoming
- [ ] Migration revert
