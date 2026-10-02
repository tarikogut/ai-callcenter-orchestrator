#!/usr/bin/env bash
#
# Startup and validation script for Turnkey AI Call Center Orchestrator Suite
# Author: Tarık Öğüt (tarik@icell.cloud)
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"

echo "=========================================================="
echo "  AI Call Center Orchestrator - Turnkey Telephony Stack   "
echo "=========================================================="

echo "[1/4] Validating FreeSWITCH configurations..."
if [ ! -f "config/freeswitch/dialplan/default.xml" ]; then
    echo "ERROR: FreeSWITCH dialplan missing!"
    exit 1
fi
if [ ! -f "config/freeswitch/autoload_configs/audio_fork.conf.xml" ]; then
    echo "ERROR: FreeSWITCH audio_fork config missing!"
    exit 1
fi
if [ ! -d "config/freeswitch/directory/default" ]; then
    echo "ERROR: FreeSWITCH directory extensions missing!"
    exit 1
fi
echo "  -> FreeSWITCH configurations OK."

echo "[2/4] Validating Kamailio SBC configuration..."
if [ ! -f "config/kamailio/kamailio.cfg" ]; then
    echo "ERROR: Kamailio configuration missing!"
    exit 1
fi
echo "  -> Kamailio configuration OK."

echo "[3/4] Validating Docker Compose configuration..."
docker compose config > /dev/null
echo "  -> docker-compose.yml syntax OK."

echo "[4/4] Starting Orchestrator Telephony Infrastructure..."
echo "Running: docker compose up -d --build"

if [ "$1" == "--dry-run" ]; then
    echo "Dry-run check completed successfully. Containers not launched."
    exit 0
fi

docker compose up -d --build

echo "=========================================================="
echo "Turnkey Stack Deployed Successfully!"
echo "  • Frontend Web Studio: http://localhost:3000"
echo "  • Orchestrator API & WS: http://localhost:8080"
echo "  • FreeSWITCH Media & ESL: localhost:5080 (SIP), localhost:8021 (ESL)"
echo "  • Kamailio SBC & Registrar: localhost:5060 (SIP UDP/TCP)"
echo "  • RTPEngine Media Proxy: localhost:22222 (Control)"
echo "=========================================================="
