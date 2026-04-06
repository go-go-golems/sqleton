---
Title: Configuring database sources
Slug: database-sources
Short: |
   There are many ways to configure database sources with sqleton. You can: 
   - pass host, port, database flags on the command line
   - load values from the environment
   - specify flags in a config file
   - use dbt profiles
Topics:
- config
- dbt
Commands:
- db
Flags:
- host
- user
- database
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

## Database source

A database source consists of the following variables:

- type: `mysql`, `postgres`, `sqlite`, `duckdb`, or another supported driver alias
- hostname
- port
- username
- password
- database
- schema (optional)
- dsn / driver (optional advanced override)

These values are combined to create a connection string that
is passed to the `sqlx` package for connection.

For server databases such as MySQL and PostgreSQL, `host`, `port`, `user`, and
`password` are usually the primary inputs. For file-based engines such as SQLite
and DuckDB, the `database` value is usually a filesystem path or an empty string
for an in-memory database.

To test a connection, you can use the `db ping` command:

``` 
❯ export SQLETON_PASSWORD=foobar
❯ sqleton db test --host localhost --port 3336 --user root    
Connection successful
```

## Command line flags

You can pass the following flags for configuring a source

      -D, --database string            Database name or database file path                               
      -H, --host string                Database host                                                      
      -p, --password string            Database password                                                  
      -P, --port int                   Database port (default 3306)                                       
      -s, --schema string              Database schema (when applicable)                                  
      -t, --type string                Database type (mysql, postgres, sqlite, duckdb, etc.) (default "mysql")
      -u, --user string                Database user                                                     
          --dsn string                 Database DSN override                                              
          --driver string              Database driver override                                           

## dbt support

[dbt](http://getdbt.com) is a great tool to run analytics against SQL databases.
In order to make it easy to reuse the configurations set by dbt, we allow loading
dbt profile files.

In order to reuse dbt profiles, you can use the following flags:

          --use-dbt-profiles           Use dbt profiles.yml to connect to databases                       
          --dbt-profile string         Name of dbt profile to use (default: default) (default "default")  
          --dbt-profiles-path string   Path to dbt profiles.yml (default: ~/.dbt/profiles.yml)            

What we call "profile" is actually a combination of profile name and output name. 
You can refer to a specific output `prod` of a profile `production` by using `production.prod`.

To get an overview of available dbt profiles, you can use the `db ls` command:

```
❯ sqleton db ls --fields name,hostname,port,database
+---------------------+-----------+-------+-------------------+
| name                | hostname  | port  | database          |
+---------------------+-----------+-------+-------------------+
| localhost.localhost | localhost | 3336  | ttc_analytics     |
| ttc.prod            | localhost | 50393 | ttc_analytics     |
| prod.prod           | localhost | 50393 | ttc_analytics     |
| dev.dev             | localhost | 50392 | ttc_dev_analytics |
+---------------------+-----------+-------+-------------------+
```


## Environment variables

All the flags mentioned above can also be set through environment variables, prefaced
with `SQLETON_` and with `-` replaced by `_`.

For example, to replace `--host localhost --port 1234 --user manuel`, use the following
environment variables:

```dotenv
SQLETON_USER=manuel
SQLETON_PORT=1234
SQLETON_HOST=localhost
```

This also applies to the dbt flags.

## Configuration

You can store all these values in a file called `config.yml`.
sqleton will look in the following locations (in that order)

- .
- $HOME/.sqleton
- /etc/sqleton

Flags and environment variables will take precedence.

The config file is a simple yaml file with the variables set:

```yaml
type: mysql
host: localhost
port: 3336
user: root
password: somewordpress
schema: wp
database: wp
```

## DuckDB file-query workflow

DuckDB is slightly different from MySQL and PostgreSQL. You connect sqleton to a
DuckDB engine, and then use SQL to read files directly.

Use an in-memory DuckDB instance when you want to inspect files ad hoc:

```bash
sqleton query --db-type duckdb --database '' \
  "SELECT * FROM read_csv_auto('./data/*.csv') LIMIT 10"
```

You can also point sqleton at a persistent DuckDB database file:

```bash
sqleton query --db-type duckdb --database ./analytics.duckdb \
  "SELECT * FROM my_table LIMIT 10"
```

To query raw files directly, keep the file path or glob inside the SQL itself:

```bash
# JSON arrays
sqleton query --db-type duckdb --database '' \
  "SELECT user_id, COUNT(*)
   FROM read_json_auto('./events/*.json', format='array')
   GROUP BY user_id"

# CSV files
sqleton query --db-type duckdb --database '' \
  "SELECT region, SUM(amount)
   FROM read_csv_auto('./exports/*.csv')
   GROUP BY region"

# Parquet files
sqleton query --db-type duckdb --database '' \
  "SELECT product, SUM(revenue)
   FROM read_parquet('./warehouse/*.parquet')
   GROUP BY product"
```

The important distinction is that the DuckDB database connection is configured by
`--db-type duckdb` and `--database ...`, while the external files are referenced
inside the SQL using DuckDB functions such as `read_json_auto`, `read_csv_auto`,
and `read_parquet`.