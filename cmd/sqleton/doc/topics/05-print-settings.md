---
Title: Printing connection settings
Slug: print-settings
Short: |
  You can print out the currently configured connection settings for quick
  reuse in an env file or other configuration file.
Topics:
  - settings
Commands:
  - print-env
  - print-evidence-settings
  - print-settings
IsTemplate: false
IsTopLevel: true
ShowPerDefault: false
SectionType: GeneralTopic
---

## Printing settings

The connection settings for sqleton can be passed in through a variety of ways.
They can be passed as environment flags, as command line flags, as a config file,
and event as a dbt profile.

Sometimes, these credentials need to be partially or fully exported for use in another
application (say, a docker-compose file, an env file, a JSON for a cloud deploy).
In order to facilitate the often tedious process, sqleton provides the following commands
to make a developer's life easier.

### `sqleton db print-env`

This command prints out the connection settings as environment variables. It is
useful for quickly exporting the settings to an env file.

```
❯ sqleton db print-env
SQLETON_TYPE=mysql
SQLETON_HOST=localhost
SQLETON_PORT=3306
SQLETON_DATABASE=sqleton
SQLETON_USER=sqleton
SQLETON_PASSWORD=secret
SQLETON_SCHEMA=bones
SQLETON_USE_DBT_PROFILES=
SQLETON_DBT_PROFILES_PATH=
SQLETON_DBT_PROFILE=yolo.bolo
```

Note that this will print out both connection settings and dbt settings. The dbt settings override the 
connection settings. It is up to you to clean the output up.

You can specify a different env variable name prefix.

```
❯ sqleton db print-env --env-prefix DB_
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_DATABASE=sqleton
DB_USER=sqleton
DB_PASSWORD=secret
DB_SCHEMA=bones
DB_USE_DBT_PROFILES=
DB_DBT_PROFILES_PATH=
DB_DBT_PROFILE=yolo.bolo
```

Furthermore, you can add `export ` at the beginning of each line by passing in the `--envrc` flag.
This makes it easier to use in `.envrc` files for the `direnv` tool.

```
❯ sqleton db print-env --envrc
export SQLETON_TYPE=mysql
export SQLETON_HOST=localhost
export SQLETON_PORT=3306
export SQLETON_DATABASE=sqleton
export SQLETON_USER=sqleton
export SQLETON_PASSWORD=secret
export SQLETON_SCHEMA=bones
export SQLETON_USE_DBT_PROFILES=
export SQLETON_DBT_PROFILES_PATH=
export SQLETON_DBT_PROFILE=yolo.bolo
```

### `sqleton db print-settings`

For a more flexible output, you can use the `print-settings` command. This command uses the glazed
library to output the connection settings, which allows you to specify the output format.

Per default, it will output the connection settings as a single row as a human readable table.

```
❯ sqleton db print-settings 
+---------+----------+-------+------------+-----------------+----------------+--------+----------+-----------+------+
| user    | password | type  | dbtProfile | dbtProfilesPath | useDbtProfiles | schema | database | host      | port |
+---------+----------+-------+------------+-----------------+----------------+--------+----------+-----------+------+
| sqleton | secret   | mysql | yolo.bolo  |                 | false          | bones  | sqleton  | localhost | 3306 |
+---------+----------+-------+------------+-----------------+----------------+--------+----------+-----------+------+
```

You can now easily print it as JSON or YAML.

``` 
❯ sqleton db print-settings --output yaml
- database: sqleton
  dbtProfile: yolo.bolo
  dbtProfilesPath: ""
  host: localhost
  password: secret
  port: 3306
  schema: bones
  type: mysql
  useDbtProfiles: false
  user: sqleton
```

You can also output each setting as a single row, for example as CSV.

```
❯ sqleton db print-settings --output csv --individual-rows
value,name
localhost,host
3306,port
sqleton,database
sqleton,user
secret,password
mysql,type
bones,schema
yolo.bolo,dbtProfile
false,useDbtProfiles
,dbtProfilesPath
```

Here too, you can specify printing things out as environment variables.

``` 
❯ sqleton db print-settings --use-env-names --individual-rows
+---------------------------+-----------+
| name                      | value     |
+---------------------------+-----------+
| SQLETON_HOST              | localhost |
| SQLETON_PORT              | 3306      |
| SQLETON_DATABASE          | sqleton   |
| SQLETON_USER              | sqleton   |
| SQLETON_PASSWORD          | secret    |
| SQLETON_TYPE              | mysql     |
| SQLETON_SCHEMA            | bones     |
| SQLETON_DBT_PROFILE       | yolo.bolo |
| SQLETON_USE_DBT_PROFILES  | false     |
| SQLETON_DBT_PROFILES_PATH |           |
+---------------------------+-----------+
```

The environment variables prefix can be configured as well.

``` 
❯ sqleton db print-settings --individual-rows --with-env-prefix DB_ 
+----------------------+-----------+
| name                 | value     |
+----------------------+-----------+
| DB_HOST              | localhost |
| DB_PORT              | 3306      |
| DB_DATABASE          | sqleton   |
| DB_USER              | sqleton   |
| DB_PASSWORD          | secret    |
| DB_TYPE              | mysql     |
| DB_SCHEMA            | bones     |
| DB_DBT_PROFILE       | yolo.bolo |
| DB_USE_DBT_PROFILES  | false     |
| DB_DBT_PROFILES_PATH |           |
+----------------------+-----------+
```

