longshoreman
============

Remote docker control utility. Currently very limited.

## Using

Longshoreman is currently limited to three functions, `repull`,
`restart` and `deploy` (which is a `repull` followed by a `restart`).  When
multiple hosts are provided the `repull` is done in parallel while the
`restart` is done in serial to minimize downtime. The `deploy` command
curently waits for all of the `repull`s to finish before starting, but this
may change in th future.

```
# One host
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command deploy
# Two hosts
$ longshoreman -image ehazlett/memcached -hosts 10.0.0.174:4243,10.0.0.175:4243 -command deploy
```

## Building

Clone the repository and build with:

    $ make

If you are on a Debian based Linux (like Ubuntu) you can also build a `.deb` package:

    # Build longshoreman.deb
    $ make dpkg
    # Build and install it
    $ make dpkg-install

## Installing
