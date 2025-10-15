#!/bin/bash
# shellcheck disable=SC1091,SC2164,SC2034,SC1072,SC1073,SC1009

# OpenVPN client removal script
# Usage: ./remove.sh <client_name> [config_file_path]
# Example: ./remove.sh myclient
# Example: ./remove.sh myclient /path/to/myclient.ovpn

set -e

function isRoot() {
	if [ "$EUID" -ne 0 ]; then
		return 1
	fi
}

function showUsage() {
	echo "Usage: $0 <client_name> [config_file_path]"
	echo ""
	echo "Arguments:"
	echo "  client_name      Name of the client to remove"
	echo "  config_file_path Optional: path to the client config file to delete"
	echo ""
	echo "Examples:"
	echo "  $0 myclient"
	echo "  $0 myclient /path/to/myclient.ovpn"
	echo ""
	echo "To list all clients, run: $0 --list"
	exit 1
}

function listClients() {
	echo "📋 Available OpenVPN clients:"
	echo ""
	
	local NUMBEROFCLIENTS
	NUMBEROFCLIENTS=$(tail -n +2 /etc/openvpn/easy-rsa/pki/index.txt | grep -c "^V" || echo "0")
	
	if [[ $NUMBEROFCLIENTS == '0' ]]; then
		echo "   No clients found."
		return 0
	fi
	
	tail -n +2 /etc/openvpn/easy-rsa/pki/index.txt | grep "^V" | cut -d '=' -f 2 | nl -s ') '
}

function checkOpenVPNInstalled() {
	if [[ ! -e /etc/openvpn/server.conf ]]; then
		echo "❌ OpenVPN server is not installed!" >&2
		echo "Please run the main installation script first." >&2
		exit 1
	fi
}

function removeClient() {
	local CLIENT="$1"
	local CONFIG_PATH="$2"
	
	# Check if client exists
	local CLIENTEXISTS
	CLIENTEXISTS=$(tail -n +2 /etc/openvpn/easy-rsa/pki/index.txt | grep -c -E "/CN=$CLIENT\$" || true)
	
	if [[ $CLIENTEXISTS == '0' ]]; then
		echo "❌ Client '$CLIENT' not found!" >&2
		exit 1
	fi
	
	cd /etc/openvpn/easy-rsa/ || {
		echo "❌ Cannot access /etc/openvpn/easy-rsa/" >&2
		exit 1
	}
	
	# Revoke client certificate
	./easyrsa --batch revoke "$CLIENT" >/dev/null 2>&1
	
	# Generate new CRL
	EASYRSA_CRL_DAYS=3650 ./easyrsa gen-crl >/dev/null 2>&1
	
	# Remove certificate files (moved to revoked directory by easyrsa revoke)
	# The revoke command already moves files to revoked directory, but we can clean up additional files
	
	# Remove any remaining PKCS files
	rm -f "/etc/openvpn/easy-rsa/pki/$CLIENT.creds" 2>/dev/null || true
	rm -f "/etc/openvpn/easy-rsa/pki/inline/$CLIENT.inline" 2>/dev/null || true
	
	# Remove duplicate certificate by serial (if exists)
	local SERIAL
	SERIAL=$(grep "^V.*CN=$CLIENT" /etc/openvpn/easy-rsa/pki/index.txt | cut -d'=' -f1 | cut -d'/' -f2 2>/dev/null || true)
	if [[ -n "$SERIAL" ]]; then
		rm -f "/etc/openvpn/easy-rsa/pki/certs_by_serial/$SERIAL.pem" 2>/dev/null || true
	fi
	
	# Update server CRL
	rm -f /etc/openvpn/crl.pem
	cp /etc/openvpn/easy-rsa/pki/crl.pem /etc/openvpn/crl.pem
	chmod 644 /etc/openvpn/crl.pem
	
	# Remove client configuration files
	if [[ -n "$CONFIG_PATH" ]]; then
		# Remove specific config file if path provided
		if [[ -f "$CONFIG_PATH" ]]; then
			rm -f "$CONFIG_PATH" 2>/dev/null || true
		fi
	else
		# Search for config files only in script directory
		local SCRIPT_DIR
		SCRIPT_DIR="$(dirname "$0")"
		rm -f "$SCRIPT_DIR/$CLIENT.ovpn" 2>/dev/null || true
	fi
	
	# Remove client from IP pool
	sed -i "/^$CLIENT,.*/d" /etc/openvpn/ipp.txt 2>/dev/null || true
	
	# Backup index file
	cp /etc/openvpn/easy-rsa/pki/index.txt{,.bk} 2>/dev/null || true
}

# Main execution
if [[ $# -lt 1 || $# -gt 2 ]]; then
	showUsage
fi

if [[ "$1" == "--list" || "$1" == "-l" ]]; then
	if ! isRoot; then
		echo "❌ This script must be run as root (use sudo) to list clients" >&2
		exit 1
	fi
	checkOpenVPNInstalled
	listClients
	exit 0
fi

if ! isRoot; then
	echo "❌ This script must be run as root (use sudo)" >&2
	exit 1
fi

checkOpenVPNInstalled

CLIENT_NAME="$1"
CONFIG_PATH="${2:-}"

removeClient "$CLIENT_NAME" "$CONFIG_PATH"
