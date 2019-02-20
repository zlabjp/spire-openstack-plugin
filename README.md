# spire-openstack-plugin
This repository contains the OpenStack based SPIRE plugins.


## Node Attestor 'openstack-iid' Plugin

The `openstack_iid` attestor is a plugin for the SPIRE Agent and SPIRE Server that allows SPIRE to automatically attest instances using the OpenStack Instance Metadata API.

### Documents

[Plugin Documents](doc/openstack-iid-attestor.md)

### Diagram

![openstack-iid-attestor-flow](images/openstack-iid-attestor-flow.png)

## Node Resolver 'openstack-iid' Plugin

The `openstack_iid` resolver plugin resolves OpenStack IID-based SPIFFE ID into a set of selectors.

### Documents

[Plugin Documents](doc/openstack-iid-resolver.md)

### Diagram

![openstack-iid-resolver-flow](images/openstack-iid-resolver-flow.png)
