# OpenStack IID Resolver
This Plugin *Must be used in conjunction with the openstack_iid node attestor plugin*.

The `openstack_iid` resolver plugin resolves OpenStack IID-based SPIFFE ID into a set of selectors.

## Selectors

| Selector            | Example                                           | Description                                                      |
| ------------------- | ------------------------------------------------- | ---------------------------------------------------------------- |
| Security Group ID   | `sg:id:sg-1234567`                                | The id of the security group the instance belongs to             |
| Security Group Name | `sg:name:default`                                 | The name of the security group the instance belongs to           |
| Custom Meta Data    | `meta:role:web`, `meta:env:dev`                   | The key=value pairs of the custom metadata[^1] that the instance has. `meta:{key}:{value}` |

 All of the selectors have the type `openstack_iid`.

 [^1]: https://developer.openstack.org/api-guide/compute/server_concepts.html#server-metadata

## Configuration

| key | type | required | description | default |
|:----|:-----|:---------|:------------|:--------|
| cloud_name | string | ✓ | Name of cloud entry in clouds.yaml to use | |
| custom_meta_data | bool   |  | Make Selector of Custom Meta Data if true | false |
| meta_data_keys   | array  |  | If `custom_meta_data` is **true**, the Selector is generated using the specified keys. If it is empty, use all entries | |

A sample configuration:

```
    NodeResolver "openstack_iid" {
        plugin_cmd = "/path/to/binary"
        plugin_checksum = "(SHOULD) sha256 of the plugin binary"
        plugin_data {
             cloud_name = "test"
             custom_meta_data = true
             meta_data_keys = ["env", "role"]
        }
    }
```
