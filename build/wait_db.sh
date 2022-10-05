#!/bin/sh

env
n=20
while :
do
  if select=$(echo 'SELECT 1' | psql postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}/${POSTGRES_DB} --quiet --no-align --tuples-only ) && [ ${select} = '1' ];
  then
    echo "DB ready"
    exit 0;
  fi;
  sleep 1
  n=$((n-1))
  if [[ $n -eq 0 ]]; then
    echo "DB not ready"
    exit 1
  fi
done