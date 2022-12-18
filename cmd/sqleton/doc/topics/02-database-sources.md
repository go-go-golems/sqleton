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

- type: postgresql, mysql
- hostname
- port
- username
- password
- database
- schema (optional)

These values are combined to create a connection string that
is passed to the `sqlx` package for connection.

> TODO(2022-12-18): support for dsn/driver flags, sqlite connection are planned
  See: 
  https://github.com/wesen/sqleton/issues/19 - add sqlite support
  https://github.com/wesen/sqleton/issues/21 - add dsn/driver flags

To test a connection, you can use the `db ping` command:

``` 
❯ export SQLETON_PASSWORD=foobar
❯ sqleton db test --host localhost --port 3336 --user root    
Connection successful
```

## Command line flags

You can pass the following flags for configuring a source

      -D, --database string            Database name                                                      
      -H, --host string                Database host                                                      
      -p, --password string            Database password                                                  
      -P, --port int                   Database port (default 3306)                                       
      -s, --schema string              Database schema (when applicable)                                  
      -t, --type string                Database type (mysql, postgres, etc.) (default "mysql")            
      -u, --user string                Database user                 

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

