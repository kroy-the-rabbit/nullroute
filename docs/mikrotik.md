# MikroTik BGP Integration

This is a practical baseline for RouterOS v7.

## RouterOS side

1. Create a BGP template and connection to your NullRoute speaker.
2. Accept only the blackhole community you expect.
3. Set route type to blackhole for those routes.

Example (adjust IP/AS values):

```routeros
/routing/filter/rule
add chain=from-nullroute rule="if (bgp-communities includes 65535:666) { set blackhole yes; accept }"
add chain=from-nullroute rule="reject"

/routing/bgp/template
add name=nullroute-template as=65001 router-id=10.255.255.1 routing-table=main

/routing/bgp/connection
add name=nullroute-peer remote.address=10.255.255.10 .as=65010 local.role=ebgp templates=nullroute-template in.filter=from-nullroute
```

## GoBGP side

- Local AS should match your NullRoute config (`65010` in examples)
- Neighbor peer-AS should match MikroTik (`65001` in examples)

## Validation

On MikroTik:

```routeros
/routing/route/print where blackhole=yes
```

On NullRoute node:

```bash
gobgp neighbor
gobgp global rib -a ipv4
gobgp global rib -a ipv6
```
