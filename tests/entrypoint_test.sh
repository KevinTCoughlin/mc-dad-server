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

# A worlds volume from an image that mounted it at /minecraft/world holds the
# world at its root. It moves into worlds/<level-name> inside the same volume.
worlds="${work_dir}/worlds"
mkdir -p "${worlds}/dimensions/minecraft/overworld" "${worlds}/lost+found"
echo level > "${worlds}/level.dat"
echo hidden > "${worlds}/.hidden"
(
    cd "${work_dir}"
    WORLDS_DIR="${worlds}" prepare_world_container
    [[ "${WORLD_ARGS[*]}" == "--world-container ${worlds}" ]]
)
[[ -f "${worlds}/world/level.dat" ]]
[[ -f "${worlds}/world/.hidden" ]]
[[ -d "${worlds}/world/dimensions/minecraft/overworld" ]]
[[ -d "${worlds}/lost+found" ]]
[[ ! -e "${worlds}/level.dat" ]]
[[ ! -e "${worlds}/.mc-dad-world-migration" ]]

# Running again leaves an already-moved world alone.
(cd "${work_dir}" && WORLDS_DIR="${worlds}" prepare_world_container)
[[ -f "${worlds}/world/level.dat" && ! -e "${worlds}/world/world" ]]

# An interrupted move is finished on the next start.
rm -rf "${worlds}" && mkdir -p "${worlds}/.mc-dad-world-migration"
echo level > "${worlds}/.mc-dad-world-migration/level.dat"
mkdir "${worlds}/region"
(cd "${work_dir}" && WORLDS_DIR="${worlds}" prepare_world_container)
[[ -f "${worlds}/world/level.dat" && -d "${worlds}/world/region" ]]

# No worlds mount (an old compose file): no world container argument.
(
    cd "${work_dir}"
    WORLDS_DIR="${work_dir}/missing" prepare_world_container
    [[ ${#WORLD_ARGS[@]} -eq 0 ]]
)

echo "entrypoint tests passed"
