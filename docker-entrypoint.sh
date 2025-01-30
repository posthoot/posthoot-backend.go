#!/bin/sh

# 🔄 Copy environment variables to .env file
printenv > .env

# ⏳ Wait for dependencies to be ready
echo "🔍 Waiting for Redis..."
while ! nc -z redis 6379; do
  sleep 1
done
echo "✅ Redis is up!"

echo "🔍 Waiting for PostgreSQL..."
while ! nc -z postgres 5432; do
  sleep 1
done
echo "✅ PostgreSQL is up!"

# 👥 Create PostgreSQL role and database permissions
echo "🔍 Creating PostgreSQL role..."
PGPASSWORD=${POSTGRES_PASSWORD} psql -h postgres -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c "CREATE ROLE ${POSTGRES_USER} WITH LOGIN PASSWORD '${POSTGRES_PASSWORD}' CREATEDB;"
PGPASSWORD=${POSTGRES_PASSWORD} psql -h postgres -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c "GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB} TO ${POSTGRES_USER};"
echo "✅ PostgreSQL role creation completed!"

# 🚀 Execute the main application
exec "/app/posthoot"
