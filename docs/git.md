# Local Docker-Based Git Server
This server uses a local reverse proxy so it can the the remote is drop in replaceable as it serves using `http://localhost` instead of `git://locahost`.

1. Have 'git-server' resolve to 'localhost'

Edit `/etc/hosts` so localhost and git-server resolve to same thing.

```
127.0.0.1   localhost git-server
```

See if it works,
```
ping git-server
```

If it doesn't, save and flush, if on Mac. Otherwise on Linux, restart you laptop.

```
sudo killall -HUP mDNSResponder
```

Initialize bare repo

```
docker exec git-server git init --bare /srv/git/<project>.git
```

Add it as your remote

```
git remote add origin http://localhost/<project>.git
```

Push, pull and clone as normal.
