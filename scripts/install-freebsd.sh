#!/usr/bin/env bash

set -e

# Default values
SERVER_ADDR=""
TOKEN=""
VERSION="latest"
REPO="awandataindonesia/cekping-agent"
SECURE="false"

# Parse arguments
while getopts "b:t:s:v:S" opt; do
  case $opt in
    b) BRANCH="$OPTARG"
    ;;
    t) TOKEN="$OPTARG"
    ;;
    s) SERVER_ADDR="$OPTARG"
    ;;
    v) VERSION="$OPTARG"
    ;;
    S) SECURE="true"
    ;;
    \?) echo "Invalid option -$OPTARG" >&2
    exit 1
    ;;
  esac
done

if [ "$VERSION" == "latest" ] && [ ! -z "$BRANCH" ] && [ "$BRANCH" != "main" ]; then
    VERSION="$BRANCH"
fi

if [ -z "$TOKEN" ]; then
    echo "Error: Token is required. Use -t <token>"
    exit 1
fi

if [ -z "$SERVER_ADDR" ]; then
    echo "Error: Server address is required. Use -s <host:port>"
    exit 1
fi

# Detect Arch
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

BINARY_NAME="cekping-agent-freebsd-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
if [ "$VERSION" == "latest" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"
fi

echo "Detected OS: freebsd ($ARCH)"

# Require root for FreeBSD installation
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: FreeBSD installation requires root privileges"
    exit 1
fi

stop_service() {
    if service cekping-agent onestatus >/dev/null 2>&1; then
        echo "Stopping existing CekPing Agent service..."
        service cekping-agent onestop || true
    fi
}

stop_service

# Create directory for agent
# /usr/local is the standard location for third-party software on FreeBSD
echo "Creating directory /usr/local/etc/cekping-agent..."
mkdir -p /usr/local/etc/cekping-agent

# Download binary agent
echo "Downloading CekPing Agent from $DOWNLOAD_URL..."
if ! curl -f -L -o /usr/local/etc/cekping-agent/cekping-agent "$DOWNLOAD_URL"; then
    echo "Error: Failed to download agent binary from $DOWNLOAD_URL"
    exit 1
fi
chmod +x /usr/local/etc/cekping-agent/cekping-agent

echo "Creating FreeBSD rc.d service..."

# Check folder rc.d are available
if [ ! -d "/usr/local/etc/rc.d" ]; then
        mkdir -p "/usr/local/etc/rc.d"
fi

cat <<'EOF' > /usr/local/etc/rc.d/cekping-agent
#!/bin/sh

# PROVIDE: cekping_agent
# REQUIRE: NETWORKING
# BEFORE: LOGIN
# KEYWORD: shutdown

. /etc/rc.subr

name="cekping_agent"
rcvar="cekping_agent_enable"

load_rc_config $name
: ${cekping_agent_enable:="NO"}

agent_bin="/usr/local/etc/cekping-agent/cekping-agent"

pidfile="/var/run/${name}.pid"
logfile="/var/log/cekping-agent.log"
procname="${agent_bin}"

command="/usr/sbin/daemon"
command_args="-p ${pidfile} -o ${logfile} ${agent_bin}"

start_precmd="cekping_precheck"

# Load config
if [ -f /usr/local/etc/cekping-agent/cekping-agent.conf ]; then
    . /usr/local/etc/cekping-agent/cekping-agent.conf
    export CEKPING_TOKEN CEKPING_SERVER CEKPING_SECURE
fi

cekping_precheck() {
    echo "Checking authentication..."

    : > "${logfile}"

    ${agent_bin} -logfile ${logfile} &
    test_pid=$!

    sleep 3

    if grep -q "Authentication failed" "${logfile}"; then
        kill ${test_pid} 2>/dev/null
        wait ${test_pid} 2>/dev/null
        echo "ERROR: Authentication failed (invalid token)"
        return 1
    fi

    kill ${test_pid} 2>/dev/null
    wait ${test_pid} 2>/dev/null

    echo "Authentication OK"
    return 0
}

run_rc_command "$1"
EOF

chmod +x /usr/local/etc/rc.d/cekping-agent

# Create configuration file
echo "Creating configuration file /usr/local/etc/cekping-agent/cekping-agent.conf..."
cat <<EOF > /usr/local/etc/cekping-agent/cekping-agent.conf
CEKPING_TOKEN="${TOKEN}"
CEKPING_SERVER="${SERVER_ADDR}"
CEKPING_SECURE="${SECURE}"
EOF

# Enable service via rc.conf
echo "Enabling service in rc.conf..."
sysrc cekping_agent_enable="YES"

# Start service
echo "Starting service..."
service cekping-agent start

# Check if service started successfully
sleep 2
if service cekping-agent onestatus >/dev/null 2>&1; then
    echo ""
    echo "CekPing Agent installed and started successfully!"
    echo "Connected to: $SERVER_ADDR"
else
    echo ""
    echo "WARNING: Service failed to start. Check authentication or connection to server $SERVER_ADDR"
    echo "Please verify your token in /usr/local/etc/cekping-agent/cekping-agent.conf"
fi

echo ""
echo "Service management:"
echo "  sudo service cekping-agent start|stop|restart|status"
echo "  sudo sysrc cekping_agent_enable=YES|NO"
