#!/bin/bash
set -e
echo "🛠️ Creating user and database 🗄️"

export PGPASSWORD=${POSTGRES_PASSWORD}
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	CREATE USER superhuman;
	CREATE DATABASE posthoot;
	ALTER USER superhuman WITH PASSWORD 'Ge+vBXEUctOwWNrO';
	GRANT ALL PRIVILEGES ON DATABASE posthoot TO superhuman;
EOSQL

echo "🛠️ User and database created 🗄️"