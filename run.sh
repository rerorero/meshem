#!/bin/bash
set -eux

go get github.com/golang/dep/cmd/dep
dep ensure
go install ./src/meshemctl

# create VMs
vagrant up

# provision control-plane
pushd ./ansible
pip install -r requirements.txt
ansible-playbook -i inventories/vagrant site.yml
popd

# provision example services with envoy proxy
pushd ./examples/ansible
pip install -r requirements.txt
ansible-playbook -i vagrant data-plane.yml
export MESHEM_CTLAPI_ENDPOINT="http://192.168.34.61:8091"
meshemctl svc apply app -f ./meshem-conf/app.yaml
meshemctl svc apply front -f ./meshem-conf/front.yaml
ansible-playbook -i vagrant site.yml -e "front_app_endpoint=http://127.0.0.1:9001/"
popd
