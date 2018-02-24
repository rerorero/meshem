docker-compose example
======
This compose file is used for quick testing in local.

#### Requirements
- go
- dep

#### Deploy containers
```
# update vendor/
pushd ../../
dep ensure
popd

# run.sh starts all of the relevant containers and register example services.
./run.sh
```
