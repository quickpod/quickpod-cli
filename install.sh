#!/bin/sh

set -eu

REPO="quickpod/quickpod-cli"
BINARY_NAME="quickpod-cli"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-}"
DRY_RUN="${DRY_RUN:-0}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

command_exists() {
	command -v "$1" >/dev/null 2>&1
}

say() {
	printf '%s\n' "$*"
}

fail() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

normalize_version() {
	value="$1"
	case "$value" in
		"" ) printf '%s' "" ;;
		v* ) printf '%s' "$value" ;;
		* ) printf 'v%s' "$value" ;;
	esac
}

detect_os() {
	case "$(uname -s)" in
		Linux) printf '%s' "linux" ;;
		Darwin) printf '%s' "darwin" ;;
		*) fail "unsupported operating system: $(uname -s)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
		x86_64|amd64) printf '%s' "amd64" ;;
		aarch64|arm64) printf '%s' "arm64" ;;
		*) fail "unsupported architecture: $(uname -m)" ;;
	esac
}

download() {
	url="$1"
	destination="$2"
	if command_exists curl; then
		if [ -n "$GITHUB_TOKEN" ]; then
			curl -fsSL --retry 3 -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" -o "$destination" "$url"
		else
			curl -fsSL --retry 3 -o "$destination" "$url"
		fi
		return
	fi
	if command_exists wget; then
		if [ -n "$GITHUB_TOKEN" ]; then
			wget --header="Authorization: Bearer ${GITHUB_TOKEN}" --header="Accept: application/vnd.github+json" -qO "$destination" "$url"
		else
			wget -qO "$destination" "$url"
		fi
		return
	fi
	fail "curl or wget is required"
}

fetch_latest_version() {
	api_url="https://api.github.com/repos/${REPO}/releases/latest"
	metadata_file="$1/latest-release.json"
	if ! download "$api_url" "$metadata_file"; then
		return 1
	fi
	version_line="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$metadata_file" | head -n 1)"
	if [ -z "$version_line" ]; then
		return 1
	fi
	printf '%s' "$version_line"
}

is_repo_checkout() {
	[ -f "./go.mod" ] && [ -f "./main.go" ]
}

resolve_source_build_metadata() {
	source_version="${VERSION}"
	if [ -z "$source_version" ]; then
		source_version="v0.1.0-dev"
	fi
	source_commit=""
	if command_exists git; then
		source_commit="$(git rev-parse --short=12 HEAD 2>/dev/null || true)"
		if [ -n "$source_commit" ] && ! git diff --quiet --ignore-submodules HEAD >/dev/null 2>&1; then
			source_version="${source_version}+${source_commit}.dirty"
		elif [ -n "$source_commit" ] && [ "$source_version" = "v0.1.0-dev" ]; then
			source_version="${source_version}+${source_commit}"
		fi
	fi
	source_build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	printf '%s\n%s\n%s\n' "$source_version" "$source_commit" "$source_build_date"
}

build_from_source() {
	target_dir="$1"
	command_exists go || fail "Go is required for source fallback installation"
	metadata="$(resolve_source_build_metadata)"
	source_version="$(printf '%s' "$metadata" | sed -n '1p')"
	source_commit="$(printf '%s' "$metadata" | sed -n '2p')"
	source_build_date="$(printf '%s' "$metadata" | sed -n '3p')"
	output_path="$tmp_dir/$BINARY_NAME"
	say "No downloadable GitHub release found; building from local source checkout"
	go build \
		-trimpath \
		-buildvcs=true \
		-ldflags "-X quickpod-cli/internal/version.Version=${source_version} -X quickpod-cli/internal/version.Commit=${source_commit} -X quickpod-cli/internal/version.BuildDate=${source_build_date}" \
		-o "$output_path" \
		.
	install_binary "$output_path" "$target_dir"
	say "Installed ${BINARY_NAME} to ${target_dir}/${BINARY_NAME}"
	case ":${PATH}:" in
		*":${target_dir}:"*) ;;
		*) say "warning: ${target_dir} is not currently in PATH" ;;
	esac
	exit 0
}

default_install_dir() {
	if [ -n "${INSTALL_DIR}" ]; then
		printf '%s' "$INSTALL_DIR"
		return
	fi

	if [ -w "/usr/local/bin" ]; then
		printf '%s' "/usr/local/bin"
		return
	fi

	printf '%s' "${HOME}/.local/bin"
}

verify_checksum() {
	archive_path="$1"
	checksum_path="$2"
	archive_name="$(basename "$archive_path")"
	expected_checksum="$(awk '{print $1}' "$checksum_path")"
	[ -n "$expected_checksum" ] || fail "checksum file for ${archive_name} is empty"

	if command_exists sha256sum; then
		actual_checksum="$(sha256sum "$archive_path" | awk '{print $1}')"
	elif command_exists shasum; then
		actual_checksum="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
	else
		say "warning: sha256sum or shasum not found, skipping checksum verification"
		return
	fi

	[ "$actual_checksum" = "$expected_checksum" ] || fail "checksum verification failed for ${archive_name}"
}

install_binary() {
	source_path="$1"
	target_dir="$2"
	mkdir -p "$target_dir"
	chmod +x "$source_path"
	if command_exists install; then
		install "$source_path" "$target_dir/$BINARY_NAME"
		return
	fi
	cp "$source_path" "$target_dir/$BINARY_NAME"
	chmod +x "$target_dir/$BINARY_NAME"
}

os="$(detect_os)"
arch="$(detect_arch)"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

VERSION="$(normalize_version "$VERSION")"
if [ -z "$VERSION" ]; then
	if ! VERSION="$(fetch_latest_version "$tmp_dir")"; then
		resolved_install_dir="$(default_install_dir)"
		if is_repo_checkout; then
			build_from_source "$resolved_install_dir"
		fi
		fail "no public GitHub release is available for ${REPO}. Push a release tag first, or set GITHUB_TOKEN if the repository is private"
	fi
	VERSION="$(normalize_version "$VERSION")"
fi

archive_name="${BINARY_NAME}_${VERSION#v}_${os}_${arch}.tar.gz"
archive_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"
checksum_url="${archive_url}.sha256"
archive_path="$tmp_dir/$archive_name"
checksum_path="$tmp_dir/${archive_name}.sha256"
resolved_install_dir="$(default_install_dir)"

say "Installing ${BINARY_NAME} ${VERSION} for ${os}/${arch}"
say "Install directory: ${resolved_install_dir}"

if [ "$DRY_RUN" = "1" ]; then
	say "Dry run enabled"
	say "Archive URL: ${archive_url}"
	say "Checksum URL: ${checksum_url}"
	exit 0
fi

if ! download "$archive_url" "$archive_path"; then
	if is_repo_checkout; then
		build_from_source "$resolved_install_dir"
	fi
	fail "failed to download ${archive_url}. Ensure the GitHub release exists and is public, or set GITHUB_TOKEN if the repository is private"
fi
if ! download "$checksum_url" "$checksum_path"; then
	if is_repo_checkout; then
		build_from_source "$resolved_install_dir"
	fi
	fail "failed to download ${checksum_url}. Ensure the release checksum asset exists"
fi
verify_checksum "$archive_path" "$checksum_path"
tar -xzf "$archive_path" -C "$tmp_dir"

binary_path="$tmp_dir/$BINARY_NAME"
[ -f "$binary_path" ] || fail "release archive did not contain ${BINARY_NAME}"

install_binary "$binary_path" "$resolved_install_dir"

say "Installed ${BINARY_NAME} to ${resolved_install_dir}/${BINARY_NAME}"
case ":${PATH}:" in
	*":${resolved_install_dir}:"*) ;;
	*) say "warning: ${resolved_install_dir} is not currently in PATH" ;;
esac
