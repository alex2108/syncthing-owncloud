# syncthing-owncloud

the scanner uses the event API of syncthing to scan files in owncloud on changes

usage:
```
scanner -ocuser="owncloudUser" -occpath="/path/to/occ" -target="http://127.0.0.1:8384" 2>&1
```
For a password protected syncthing instance the apikey needs to be provided either by using `-api="syncthing-api-key"` or `-apikey-from-stdin` and then entering the apikey there (or piping it `echo "apikey" | scanner ...`). For a server where multiple users have access, using stdin is recommended to prevent seeing the apikey in the running processes.

`-target=...` is optional if the default `http://127.0.0.1:8384` is used for syncthing


In case your php needs to be run in a special way, [this](https://github.com/alex2108/syncthing-owncloud/blob/master/scanner/main.go#L92) line needs to be adjusted.
