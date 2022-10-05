# Service: template

## Getting started

1. Execute `make init-project`, it exchange dependent names.
2. Execute `make gen`, it create required dir.
3. Execute `make migadd name=init`, it create first migration

## Second step

1. Add SQL to schema
2. Execute `make gensql`, it generate repository
3. Execute `make infra-up`, it create db in docker.  !!! Need to run docker

## Next lvl

1. Add any functions to repository
2. Add processor
3. Update swagger
