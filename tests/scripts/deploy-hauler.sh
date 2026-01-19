#!/bin/bash
set -e -x
# This script DOENST WORK installs hauler as a service
# Note: the hauler binary must be installed in /usr/local/bin and data have to be synced already beforehand

# Check the data is synced already

# Capture caller info
CALLER=${SUDO_USER:-$(whoami)}

# Create the service file
cat > /etc/systemd/system/hauler-registry@.service << EOF
[Unit]
Description=Hauler OCI Registry
After=network.target
# Start the service as hauler-registry@5001_version

[Service]
Type=simple
User=${CALLER}
Group=$(id -gn ${CALLER})
Environment=STORAGE_ROOT=/home/${CALLER}/hauler
WorkingDirectory=/home/${CALLER}/hauler

# We use parameter expansion to split the port and version safely.
# For bash scripts in systemd file in combination with HEREDOC '\$\$VAR'
ExecStart=/bin/bash -c 'v="%i"; PORT="\$\${v%_*}"; VERSION="\$\${v#*_}"; exec logger PORT=\$\$PORT VERSION=\$\$VERSION'
#exec /usr/local/bin/hauler store serve registry --port "$PORT" --store "$STORAGE_ROOT/$VERSION-store" --directory "$STORAGE_ROOT/$VERSION-registry"'

# Same logic for the health check
# ExecStartPost=/bin/bash -c 'v="%i"; PORT="${v%-*}"; echo "Waiting for Hauler on port $PORT..."; until curl -s -f http://localhost:"$PORT"/v2/ > /dev/null; do sleep 1; done'

Restart=always
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
