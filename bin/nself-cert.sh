#!/bin/bash

# nself-cert.sh - Manage TLS certificates for development and production

set -e

# ----------------------------
# Helper Functions
# ----------------------------

echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

detect_os() {
  unameOut="$(uname -s)"
  case "${unameOut}" in
      Linux*)     OS=Linux;;
      Darwin*)    OS=Mac;;
      CYGWIN*)    OS=Cygwin;;
      MINGW*)     OS=MinGw;;
      *)          OS="UNKNOWN"
  esac
  echo "${OS}"
}

install_mkcert_mac() {
  if ! command -v brew >/dev/null 2>&1; then
    echo_error "Homebrew is not installed. Please install it from https://brew.sh/"
    exit 1
  fi
  echo_info "Installing mkcert using Homebrew..."
  brew install mkcert nss >/dev/null 2>&1
}

install_mkcert_linux() {
  echo_info "Installing mkcert on Linux..."
  sudo apt-get update -qq >/dev/null 2>&1
  sudo apt-get install -y libnss3-tools wget >/dev/null 2>&1
  wget -q -O mkcert https://github.com/FiloSottile/mkcert/releases/latest/download/mkcert-v1.4.4-linux-amd64
  chmod +x mkcert
  sudo mv mkcert /usr/local/bin/ >/dev/null 2>&1
}

generate_local_certs() {
  OS=$(detect_os)
  
  if ! command -v mkcert >/dev/null 2>&1; then
    echo_info "mkcert not found. Installing mkcert..."
    if [ "$OS" == "Mac" ]; then
      install_mkcert_mac
    elif [ "$OS" == "Linux" ]; then
      install_mkcert_linux
    else
      echo_error "Unsupported OS: $OS. Please install mkcert manually."
      exit 1
    fi
  fi

  echo_info "Setting up mkcert local CA..."
  mkcert -install >/dev/null 2>&1

  CERT_DIR=".traefik/certs"
  mkdir -p "$CERT_DIR"

  for HOST in "$@"; do
    CERT_FILE="${CERT_DIR}/${HOST}.pem"
    KEY_FILE="${CERT_DIR}/${HOST}-key.pem"

    if [ -f "$CERT_FILE" ] && [ -f "$KEY_FILE" ]; then
      echo_info "Certificate for ${HOST} already exists. Skipping generation."
    else
      echo_info "Generating certificate for ${HOST}..."
      mkcert -cert-file "$CERT_FILE" -key-file "$KEY_FILE" "$HOST" >/dev/null 2>&1
      echo_info "Certificate for ${HOST} generated."
    fi
  done
}

configure_letsencrypt() {
  echo_info "Configuring Traefik for Let's Encrypt..."

  TRAEFIK_YML="traefik.yml"

  # Check if Let's Encrypt configuration already exists
  if grep -q "certificatesResolvers:" "$TRAEFIK_YML"; then
    echo_info "Let's Encrypt configuration already exists in $TRAEFIK_YML."
  else
    echo_info "Appending Let's Encrypt configuration to $TRAEFIK_YML."
    cat <<EOF >> "$TRAEFIK_YML"

certificatesResolvers:
  letsencrypt:
    acme:
      email: ${LETSENCRYPT_EMAIL}
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
EOF
  fi

  # Ensure acme.json file exists and has correct permissions
  if [ ! -f "./letsencrypt/acme.json" ]; then
    mkdir -p ./letsencrypt
    touch ./letsencrypt/acme.json
    chmod 600 ./letsencrypt/acme.json
    echo_info "Created acme.json for Let's Encrypt storage."
  fi
}

# ----------------------------
# Main Logic
# ----------------------------

ENVIRONMENT="${ENV:-dev}"  # Default to development if ENV is not set

if [ "$ENVIRONMENT" == "prod" ]; then
  # Production environment: Configure Let's Encrypt
  configure_letsencrypt
else
  # Development environment: Generate local certificates using mkcert
  if [ "$#" -lt 1 ]; then
    echo_error "No hostnames provided for certificate generation."
    echo "Usage: $0 hostname1 [hostname2 ...]"
    exit 1
  fi
  generate_local_certs "$@"
fi

echo_info "Certificate management completed for environment: ${ENVIRONMENT}"
