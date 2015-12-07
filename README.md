# scanner


the scanner uses the event API of syncthing to scan files in owncloud on changes

usage:
```
scanner -ocuser="owncloudUser" -stfolder="cloud" -occpath="/path/to/occ" -target="http://127.0.0.1:8384" 2>&1
```
For a password protected syncthing instance the apikey needs to be provided either by using `-api="syncthing-api-key"` or `-apikey-from-stdin` and then entering the apikey there (or piping it `echo "apikey" | scanner ...`). For a server where multiple users have access, using stdin is recommended to prevent seeing the apikey in the running processes.

`-target=...` is optional if the default `http://127.0.0.1:8384` is used for syncthing


In case your php needs to be run in a special way, [this](https://github.com/alex2108/syncthing-owncloud/blob/master/scanner/main.go#L92) line needs to be adjusted.

# versioner

The `archive` program is used for the external versioner in syncthing. It takes three parameters, the first two are the folder path and path insidde the folder like for the external versioner, the third argument is the path to the version folder. Therefore a small script as wrapper for the external versioner is needed, for example:
```
#!/bin/bash
/path/to/archive "$1" "$2" "/path/to/owncloud/data/owncloudUser/files_versions"
```
This small script is then set for the external versioner in syncthing.

The `clean` program can be used to automatically clean out versions like the staggered versioner of syncthing does. This can be run as a cronjob. It takes the version path as first argument:
```
/path/to/clean /path/to/owncloud/data/owncloudUser/files_versions

```
