# OpenStack IID Atestor

## Overview

The OpenStack InstanceID(IID) attestor is a plugin for the SPIRE Agent and SPIRE Server that allows SPIRE to automatically attest instances using the OpenStack Instance Metadata API. Agents attested by the `openstack_iid` attestor will be issued a SPIFFE ID like `spiffe://TRUST_DOMAIN/agent/openstack_iid/PROJECT_ID/INSTANCE_ID`. This plugin requires a whitelist of ProjectID from which nodes can be attested. This also means that you shouldn't run multiple trust domains from the same OpenStack Project(**TBD**).

| Configuration           | Description                                                       | Default |
|-------------------------|-------------------------------------------------------------------|---------|
| projectid_whitelist     | List of whitelisted ProjectIDs from which nodes can be attested.  |         |


## Base SVID SPIFFE ID Format

```
spiffe://TRUST_DOMAIN/agent/openstack_iid/PROJECT_ID/INSTANCE_ID
```

## Pre-Requisites

Thie plugin requires a running SPIRE server and agent each on the OpenStack Nova Instances.

## Configuring the server plugin

https://github.com/spiffe/spire/blob/master/conf/server/server.conf

The sever plugin configuration template is as below:

```hcl
plugins {
    NodeAttestor "openstack_iid" {
        plugin_cmd = "/path/to/plugin_cmd"
        plugin_data {
            projectid_whitelist = ["123", "abc"]
        }
    }
...
```

The plugin_name should be "openstack_iid" and matches the name used in plugin config. The plugin_cmd should specify the path to the plugin binary. 

## Configuring agent plugin

https://github.com/spiffe/spire/blob/master/conf/agent/agent.conf

The agent plugin configuration template is as below:

```hcl
plugins {
    NodeAttestor "openstack_iid" {
        plugin_cmd = "/path/to/plugin_cmd"
        plugin_data {
        }
    }
...
```

The plugin_name should be "openstack_iid" and matches the name used in plugin config. The plugin_cmd should specify the path to the agent binary. 
