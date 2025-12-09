#!/bin/sh
set -e

echo "Starting application in ${APP_ENV:-development} mode..."

# Set default environment to development if not set
APP_ENV=${APP_ENV:-development}
ENV_FILE="/app/.env.${APP_ENV}"

# Check if the environment file exists
if [ -f "$ENV_FILE" ]; then
    echo "Loading environment variables from ${ENV_FILE}"
    # Export all variables from the environment file
    set -o allexport
    . "$ENV_FILE"
    set +o allexport
else
    echo "Warning: Environment file ${ENV_FILE} not found. Using default environment variables."
fi

# Run the application
exec "$@"
