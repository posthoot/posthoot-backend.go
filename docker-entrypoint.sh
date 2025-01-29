#!/bin/sh

# 🔄 Copy environment variables to .env file
printenv > .env

cat .env

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

# 🚀 Execute the main application
exec "/app/posthoot"
