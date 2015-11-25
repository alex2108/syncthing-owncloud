# syncthing-owncloud

the scanner uses the event API of syncthing to scan files in owncloud on changes

usage:
```
scanner -ocuser="owncloudUser" -occpath="/path/to/occ" -target="http://127.0.0.1:8384" -api="syncthing-api-key" 2>&1
```

In case your php needs to be run in a special way, [this](https://github.com/alex2108/syncthing-owncloud/blob/master/scanner/main.go#L90) line needs to be adjusted.
