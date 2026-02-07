#!/usr/bin/env bash

set -euxo pipefail

docker load -i deploy/build/docker-valvx-app-api.tar
docker load -i deploy/build/docker-valvx-app-web.tar

cp -RT deploy/conf conf_tmp
chown -R root:valvx conf_tmp
chmod -R g+w conf_tmp
chmod -R o-rwx conf_tmp

rsync -a conf_tmp/ /opt/valvx

cd /opt/valvx

docker compose run api migrate
docker compose up -d
docker compose exec -w /etc/caddy caddy caddy reload
