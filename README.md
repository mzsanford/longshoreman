longshoreman
============

Remote docker control utility. Currently very limited.

## Using

Longshoreman is currently limited to only some functions, the most useful of
which are `pull`, `restart` and `deploy` (which is a `pull` followed by a `restart`).
When multiple hosts are provided the `pull` is done in parallel while the
`restart` is done in serial to minimize downtime (a "rolling restart"). The
`deploy` command curently waits for all of the `pull`s to finish before starting, but this
may change in th future.

```
# One host
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command deploy
# Two hosts
$ longshoreman -image ehazlett/memcached -hosts 10.0.0.174:4243,10.0.0.175:4243 -command deploy
```

### Commands

All of these commands allow for some common arguments, which can be listed with `longshoreman -help`.
Some commonly used are `-q` for quiet mode and `-v` for verbose mode.

#### pull

Initiate a `docker pull` for the image provided. This is run in parallel on
all hosts.

```
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command pull
```

#### restart

Initiate a `docker restart` for all containers running from the image provided. This
is run serially.

```
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command restart
```

#### deploy

Initiate a `docker pull` for all containers running from the image provided followed
by a `docker restart`. The `docker pull` is run in parallel and once all of those have
finished the `docker restart` calls are done serially.

```
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command deploy
```

#### stop

Initiate a `docker stop` for all containers running from the image provided. This
is run serially.

```
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command stop
```

#### list

Get the current status of all containers running from the image provided.

```
$ longshoreman -image ehazlett/memcached -hosts 192.168.42.43:4243 -command list
2014/03/20 17:18:43 [ INFO] Starting list of ehazlett/memcached on 1 hosts
192.168.42.43:4243/ehazlett/memcached[77d41b558221099]: up (39 hours, 31 minutes)
2014/03/20 17:18:43 [ INFO] Completed list of ehazlett/memcached on 1 hosts completed
```

The `77d41b558221099` in the above output is the image id, useful for checking that
all containers running `:latest` are running the same image.

#### cat

Get the contents of a file **inside** every container running from the image provided. This
can be used for many things but the initial use case was check code versions on multiple
instances for discrepancies.

```
$ longshoreman -hosts 10.0.0.175:4243,10.0.0.176:4243,10.0.0.177:4243 -image privateregisty.com/company/appname -command cat -file /opt/appname/.git/HEAD
2014/03/21 00:21:21 [ INFO] Starting cat of privateregisty.com/company/appname on 3 hosts

10.0.0.175:4243: 5eacfb0c4fd194574009ceca3d15d5383649a5dd

10.0.0.176:4243: 5eacfb0c4fd194574009ceca3d15d5383649a5dd
2014/03/21 00:21:21 [ INFO] Completed cat of privateregisty.com/company/appname on 3 hosts

10.0.0.177:4243: 5eacfb0c4fd194574009ceca3d15d5383649a5dd
```

When run with the `-q` option the output is a bit clearer:

```
$ longshoreman -hosts 10.0.0.175:4243,10.0.0.176:4243,10.0.0.177:4243 -image privateregisty.com/company/appname -command cat -file /opt/appname/.git/HEAD -q

10.0.0.175:4243: 5eacfb0c4fd194574009ceca3d15d5383649a5dd

10.0.0.176:4243: 5eacfb0c4fd194574009ceca3d15d5383649a5dd

10.0.0.177:4243: 5eacfb0c4fd194574009ceca3d15d5383649a5dd
```

## Building

Clone the repository and build with:

    $ make

If you are on a Debian based Linux (like Ubuntu) you can also build a `.deb` package:

    # Build longshoreman.deb
    $ make dpkg

## Installing

Clone the repository and install with:

    $ make install

Or, if you're on a Debian based Linux (like Ubuntu) you can install it
via the `.deb` package either from the [releases page](https://github.com/mzsanford/longshoreman/releases)
or by building and install with:

    $ make dpkg-install
