#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="${repo_dir}/.entrypoint-test"
trap 'rm -rf "${work_dir}"' EXIT

rm -rf "${work_dir}"
mkdir "${work_dir}"
cp "${repo_dir}/configs/server.properties" "${work_dir}/server.properties"

export RCON_PASSWORD='abc\def'
export MOTD='Dads \ Kids'

# shellcheck disable=SC1091
source "${repo_dir}/entrypoint.sh"

(
    cd "${work_dir}"
    configure_server_properties
)

grep -Fqx 'rcon.password=abc\\def' "${work_dir}/server.properties"
grep -Fqx 'motd=Dads \\ Kids' "${work_dir}/server.properties"
