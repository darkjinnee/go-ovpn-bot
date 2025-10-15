#!/bin/bash
# shellcheck disable=SC1091,SC2164,SC2034,SC1072,SC1073,SC1009

# OpenVPN client addition script
# Usage: ./add.sh <client_name> [output_directory] [password_protected]
# Example: ./add.sh myclient
# Example: ./add.sh myclient /path/to/configs
# Example: ./add.sh myclient /path/to/configs password

set -e

function isRoot() {
	if [ "$EUID" -ne 0 ]; then
		return 1
	fi
}

function showUsage() {
	echo "Usage: $0 <client_name> [output_directory] [password_protected]"
	echo ""
	echo "Arguments:"
	echo "  client_name        Name for the client (alphanumeric, underscore, dash allowed)"
	echo "  output_directory   Optional: directory to save config file (default: script directory)"
	echo "  password_protected Optional: 'password' to protect the client with a password"
	echo ""
	echo "Examples:"
	echo "  $0 myclient"
	echo "  $0 myclient /path/to/configs"
	echo "  $0 myclient /path/to/configs password"
	exit 1
}

function checkOpenVPNInstalled() {
	if [[ ! -e /etc/openvpn/server.conf ]]; then
		echo "❌ OpenVPN server is not installed!"
		echo "Please run the main installation script first."
		exit 1
	fi
}

function addClient() {
	local CLIENT="$1"
	local OUTPUT_DIR="$2"
	local PASSWORD_PROTECTED="$3"
	
	# Validate client name
	if [[ ! $CLIENT =~ ^[a-zA-Z0-9_-]+$ ]]; then
		echo "❌ Invalid client name. Use only alphanumeric characters, underscores, and dashes." >&2
		exit 1
	fi
	
	# Check if client already exists
	local CLIENTEXISTS
	CLIENTEXISTS=$(tail -n +2 /etc/openvpn/easy-rsa/pki/index.txt | grep -c -E "/CN=$CLIENT\$" || true)
	if [[ $CLIENTEXISTS == '1' ]]; then
		echo "❌ Client '$CLIENT' already exists!" >&2
		exit 1
	fi
	
	# Validate and create output directory
	if [[ -n "$OUTPUT_DIR" ]]; then
		# Convert relative path to absolute path
		if [[ ! "$OUTPUT_DIR" = /* ]]; then
			OUTPUT_DIR="$(pwd)/$OUTPUT_DIR"
		fi
		if [[ ! -d "$OUTPUT_DIR" ]]; then
			if ! mkdir -p "$OUTPUT_DIR" 2>/dev/null; then
				echo "❌ Cannot create output directory: $OUTPUT_DIR" >&2
				exit 1
			fi
		fi
		if [[ ! -w "$OUTPUT_DIR" ]]; then
			echo "❌ Output directory is not writable: $OUTPUT_DIR" >&2
			exit 1
		fi
	else
		# Use script directory as default
		OUTPUT_DIR="$(dirname "$0")"
	fi
	
	cd /etc/openvpn/easy-rsa/ || {
		echo "❌ Cannot access /etc/openvpn/easy-rsa/" >&2
		exit 1
	}
	
	# Generate client certificate
	if [[ "$PASSWORD_PROTECTED" == "password" ]]; then
		EASYRSA_CERT_EXPIRE=3650 ./easyrsa --batch build-client-full "$CLIENT" >/dev/null 2>&1
	else
		EASYRSA_CERT_EXPIRE=3650 ./easyrsa --batch build-client-full "$CLIENT" nopass >/dev/null 2>&1
	fi
	
	# Use specified output directory
	local homeDir="$OUTPUT_DIR"
	
	# Get server configuration
	local SERVER_NAME
	SERVER_NAME=$(cat /etc/openvpn/easy-rsa/SERVER_NAME_GENERATED 2>/dev/null || echo "server")
	
	# Determine TLS signature type
	local TLS_SIG
	if grep -qs "^tls-crypt" /etc/openvpn/server.conf; then
		TLS_SIG="1"
	elif grep -qs "^tls-auth" /etc/openvpn/server.conf; then
		TLS_SIG="2"
	fi
	
	# Get server configuration parameters
	local PORT PROTOCOL HMAC_ALG CIPHER CC_CIPHER
	PORT=$(grep '^port ' /etc/openvpn/server.conf | cut -d " " -f 2)
	PROTOCOL=$(grep '^proto ' /etc/openvpn/server.conf | cut -d " " -f 2)
	HMAC_ALG=$(grep '^auth ' /etc/openvpn/server.conf | cut -d " " -f 2)
	CIPHER=$(grep '^cipher ' /etc/openvpn/server.conf | cut -d " " -f 2)
	CC_CIPHER=$(grep '^tls-cipher ' /etc/openvpn/server.conf | cut -d " " -f 2)
	
	# Get server IP
	local IP
	IP=$(grep '^remote ' /etc/openvpn/client-template.txt 2>/dev/null | cut -d " " -f 2 || echo "YOUR_SERVER_IP")
	
	# Generate client configuration
	
	# Create client template
	cat > "$homeDir/$CLIENT.ovpn" << EOF
client
$(if [[ $PROTOCOL == 'udp' ]]; then
	echo "proto udp"
	echo "explicit-exit-notify"
elif [[ $PROTOCOL == 'tcp' ]]; then
	echo "proto tcp-client"
fi)
remote $IP $PORT
dev tun
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
verify-x509-name $SERVER_NAME name
auth $HMAC_ALG
auth-nocache
cipher $CIPHER
tls-client
tls-version-min 1.2
tls-cipher $CC_CIPHER
ignore-unknown-option block-outside-dns
setenv opt block-outside-dns # Prevent Windows 10 DNS leak
verb 3
EOF

	# Add compression if enabled
	if grep -qs "^compress" /etc/openvpn/server.conf; then
		local COMPRESSION_ALG
		COMPRESSION_ALG=$(grep '^compress ' /etc/openvpn/server.conf | cut -d " " -f 2)
		echo "compress $COMPRESSION_ALG" >> "$homeDir/$CLIENT.ovpn"
	fi
	
	# Add certificates and keys
	{
		echo "<ca>"
		cat "/etc/openvpn/easy-rsa/pki/ca.crt"
		echo "</ca>"
		
		echo "<cert>"
		awk '/BEGIN/,/END CERTIFICATE/' "/etc/openvpn/easy-rsa/pki/issued/$CLIENT.crt"
		echo "</cert>"
		
		echo "<key>"
		cat "/etc/openvpn/easy-rsa/pki/private/$CLIENT.key"
		echo "</key>"
		
		case $TLS_SIG in
		1)
			echo "<tls-crypt>"
			cat /etc/openvpn/tls-crypt.key
			echo "</tls-crypt>"
			;;
		2)
			echo "key-direction 1"
			echo "<tls-auth>"
			cat /etc/openvpn/tls-auth.key
			echo "</tls-auth>"
			;;
		esac
	} >> "$homeDir/$CLIENT.ovpn"
	
	# Output only the path to the config file
	echo "$homeDir/$CLIENT.ovpn"
}

# Main execution
if [[ $# -lt 1 || $# -gt 3 ]]; then
	showUsage
fi

if ! isRoot; then
	echo "❌ This script must be run as root (use sudo)" >&2
	exit 1
fi

checkOpenVPNInstalled

CLIENT_NAME="$1"
OUTPUT_DIRECTORY="${2:-}"
PASSWORD_PROTECTED="${3:-}"

addClient "$CLIENT_NAME" "$OUTPUT_DIRECTORY" "$PASSWORD_PROTECTED"
