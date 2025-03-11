#!/bin/sh
# Admin tool wrapper script

echo "Using database at: $DB_PATH"
echo "Running admin tool with arguments: $@"

# Run the admin tool
./admin-tool "$@"
