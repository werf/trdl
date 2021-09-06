#!/usr/bin/env sh

# Invoking this script:
#
# curl https://raw.githubusercontent.com/werf/trdl/main/client/get.sh | sh
#
# Actions:
# - check os and arch
# - detect curl or wget
# - check bintary for latest available release of trdl binary
# - making sure trdl is executable
# - print brief usage

set -e -o nounset

http_client="curl"

detect_downloader() {
  if tmp=$(curl --version 2>&1 >/dev/null) ; then return ; fi
  if tmp=$(wget --help 2>&1 >/dev/null) ; then http_client="wget" ; return ; fi
  echo "Cannot detect curl or wget. Install one of them and run again."
  exit 2
}

# download_file URL OUTPUT_FILE_PATH
download_file() {
  if [ "${http_client}" = "curl" ] ; then
    if ! curl -Ls "$1" -o "$2" ; then
      echo "curl error for file $1"
      return 1
    fi
    return
  fi
  if [ "${http_client}" = "wget" ] ; then
    if ! wget -q -O "$2" "$1" ; then
      echo "wget error for file $1"
      return 1
    fi
  fi
}

# get_location_header URL
get_location_header() {
  if [ "${http_client}" = "curl" ] ; then
    if ! curl -s "$1" -w "%{redirect_url}" ; then
      echo "curl error for $1"
      return 1
    fi
    return
  fi
  if [ "${http_client}" = "wget" ] ; then
    if ! wget -S -q -O - "$1" 2>&1 | grep -m 1 'Location:' | tr -d '\r\n' ; then
      echo "wget error for $1"
      return 1
    fi
  fi
}

check_os_arch() {
  supported="linux-amd64 linux-arm64 darwin-amd64 darwin-arm64"

  if ! echo "${supported}" | tr ' ' '\n' | grep -q "${OS}-${ARCH}"; then
    cat <<EOF

${PROGRAM} installation is not currently supported on ${OS}-${ARCH}.

See https://github.com/werf/trdl for more information.

EOF
  fi
}

PROGRAM="trdl"
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "${ARCH}" = "x86_64" ] ; then
  ARCH="amd64"
fi

if [ "${ARCH}" = "aarch64" ] ; then
  ARCH="arm64"
fi

check_os_arch

detect_downloader

VERSION="0.1.3"
DOWNLOAD_URL="https://tuf.trdl.dev/targets/releases/${VERSION}/${OS}-${ARCH}/bin/trdl"

echo "Downloading ${DOWNLOAD_URL}..."
if ! download_file "${DOWNLOAD_URL}" trdl
then
  exit 2
fi

chmod +x "${PROGRAM}"

cat <<EOF

${PROGRAM} is now available in your current directory.

To learn more, execute:

    $ ./${PROGRAM} help

EOF
