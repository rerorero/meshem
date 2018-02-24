meshem
=======
[![Build Status](https://travis-ci.org/rerorero/meshem.svg?branch=master)](https://travis-ci.org/rerorero/meshem)


Ansible example
=======

### Requirments
- Vagrant + Virtual Box
- python

Create all VMs
```
vagrant up
```

### Launch example services without proxy
Provision example services without envoy proxy. The example consists of `app` service and `front` service, both of them provide HTTP API.
```
cd ./examples/ansible
pip install -r requirements.txt
ansible-playbook -i vagrant site.yml
```
Check the response from front application.
```
curl 192.168.34.70:8080
```

### Let's mesh'em
Provision the control plane.
```
cd ./ansible
ansible-playbook -i inventories/vagrant site.yml
```
Launch envoy proxies for each services.
```
cd ./examples/ansible
ansible-playbook -i vagrant data-plane.yml
```

#### Manage meshem
You can download a CLI binary(meshemctl) from [Github release page](https://github.com/rerorero/meshem/releases). Put the binary somewhere in your `$PATH`. meshemctl also can be built from source by running `go install github.com/rerorero/meshem/src/meshemctl` command.

Then let us regsiter the app service and front service. 
```
cd ./examples/ansible
export MESHEM_CTLAPI_ENDPOINT="http://192.168.34.61:8091"
meshemctl svc apply app -f ./meshem-conf/app.yaml
meshemctl svc apply front -f ./meshem-conf/front.yaml
```

#### Deploy services as a service mesh
```
ansible-playbook -i vagrant site.yml -e "front_app_endpoint=http://127.0.0.1:9001/"
```
```
curl 192.168.34.70:80
```


Docker example
=======
See (this doc)[examples/docker/README.md].

