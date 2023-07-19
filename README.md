# spire-openstack-plugin (experimental)
This repository contains the OpenStack based SPIRE plugins.

**Not ready for a production release**

See: [Security Consideration](doc/openstack-iid-attestor.md#security-consideration)

## Node Attestor 'openstack-iid' Plugin

The `openstack_iid` attestor is a plugin for the SPIRE Agent and SPIRE Server that allows SPIRE to automatically attest instances using the OpenStack Instance Metadata API.

### Documents

[Plugin Documents](doc/openstack-iid-attestor.md)

### Diagram

![openstack-iid-attestor-flow](images/openstack-iid-attestor-flow.png)

## LICENSE

This software is released under the MIT License.
