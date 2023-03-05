## Scanner

The scanner uses the event API of syncthing to scan files in nextcloud on changes.

Usage:

```
scanner -mapping='nextcloudUser1/nextcloudFolderName1/stfolderID1' -mapping='nextcloudUser2/nextcloudFolderName2/stfolderID2' -occpath="/path/to/occ" -target="http://127.0.0.1:8384" 2>&1
```

For a password protected syncthing instance the apikey needs to be provided either by using `-api="syncthing-api-key"` or `-apikey-from-stdin` and then entering the apikey there (or piping it `echo "apikey" | scanner ...`). For a server where multiple users have access, using stdin is recommended to prevent seeing the apikey in the running processes.

`-target=...` is optional if the default `http://127.0.0.1:8384` is used for syncthing

## Versioner

The `archive` program is used for the external versioner in syncthing. It takes three parameters, the first two are the folder path and path inside the folder like for the external versioner, the third argument is the path to the version folder. For example:

```
/path/to/archive '%FOLDER_PATH%' '%FILE_PATH%' '/path/to/nextcloud/data/nextcloudUser/files_versions/external_storage_folder_name'
```
This command is then set for the external versioner in syncthing.

The `clean` program can be used to automatically clean out versions like the staggered versioner of syncthing does. This can be run as a cronjob. It takes the version path as first argument:

```
/path/to/clean /path/to/nextcloud/data/nextcloudUser/files_versions/external_storage_folder_name

```

## Build

Clone this repo, cd into it and run the commands below to build the three binaries using docker.

```
docker run --rm -e CGO_ENABLED=0 -v "$(pwd)":/app --workdir=/app/scanner golang:alpine go build
docker run --rm -e CGO_ENABLED=0 -v "$(pwd)":/app --workdir=/app/versioner/archive golang:alpine go build
docker run --rm -e CGO_ENABLED=0 -v "$(pwd)":/app --workdir=/app/versioner/clean golang:alpine go build
```

## Deploy

This is an example deployment using docker compose. Open a terminal and cd into the directory where you've cloned this repo (and built the binaries, see above).

Create the directories where syncthing and nextcloud will store their files.

```
mkdir syncthing config data postgresql
```

Tweak the settings in the compose file to your liking (e.g. replace the postgres password).

Start the stack.

```
docker compose up -d
```

Now visit nextcloud in your browser at https://ip-or-hostname:8443 and go through the initial setup process. Don't forget to unfold the `Storage & database` dropdown, click `PostgreSQL` and fill in the following:

- Database user: nextcloud
- Database password: the password you have in your compose file
- Database name: nextcloud
- Database host: db

Now visit syncthing in your browser at https://ip-or-hostname:8384 and setup password authentication. Now go to Actions -> Settings -> Edit Folder Defaults -> File Versioning. Choose `External File Versioning` and for `Command` set: `/versioner/archive/archive '%FOLDER_PATH%' '%FILE_PATH%' '/var/syncthing/versions/%FOLDER_PATH%'`. This will ensure previous versions of files will be stored with a nextcloud versions compatible filename inside the `/var/syncthing/versions` directory.

Back in the main syncthing dashboard. Remove the default folder. Click on `Add Folder`, to create a new folder for your first user. Choose a Folder Label (e.g. user1) and copy the random id (e.g.) `2ja6j-4xtmq`. In the compose file update the `NC_ST_MAPPING_1` environment value. It is in the following format nextcloud username/nextcloud external storage folder name/syncthing folder ID. This environment variable tells the `scanner` binary which nextcloud user folder needs a rescan when there are changes inside the syncthing folder. Also copy the `Folder Path` (e.g. `/var/syncthing/user1`). We need this value for the next step. Click `Save`.

Login as admin into nextcloud and go to https://ip-or-hostname:8443/settings/users. Create two new user (user1 and user2). Then go to `Apps` (by clicking on the avatar image in the top right) or go to https://ip-or-hostname:8443/settings/apps. Scroll down and enable the `External storage support` App. Go to https://ip-or-hostname:8443/settings/admin/externalstorages and add an external storage folder.

Now we need the syncthing `Folder Path` value copied previously, paste it in the `Configuration` field.

- Folder name: sync
- External storage: Local
- Authentication: none
- Configuration: /var/syncthing/user1
- Available for: user1

Click the checkmark on the right to apply the settings.

Then make another change in the compose file to mount the `versions` directory in the appropriate location for user1. The compose file already has a commented example line in it. Uncomment `- ./syncthing/versions/var/syncthing/user1:/data/user1/files_versions/sync` and change `user1` and/or `sync` in case you chose a different username or folder name.

It is now time to enable the scanner we built before, to trigger indexing the nextcloud external storage once changes are detected by syncthing. The scanner is started thanks to [scan.sh](custom-services.d/scan.sh) and the [Custom Services](https://docs.linuxserver.io/general/container-customization#custom-services) feature offered by the linuxserver.io nextcloud image. The `scan.sh` file reads the syncthing API key from a file. But since we haven't created this file yet, it's now waiting in a loop. It checks again every 60 seconds if the file has been created. So let's do that!

Open the syncthing interface and go to Actions -> General and copy the API Key. Now go back to the terminal. Change into the `config` directory (as root user) and create the `syncthing_api_key.txt` file with the API key as the only contents. E.g. `echo "PXwMRJcafZ7fuAPvyg9tZcmouqbDJtdH" > syncthing_api_key.txt`. Change the permissions so that only the root user can read this file: `chmod 600 syncthing_api_key.txt`. The file should get picked up and the `scanner` should start.

You may now add devices in the syncthing interface and share folders with it. File changes are now picked up by `scanner` and it will trigger a rescan. Look for `Start PHP scan` in the nextcloud container logs.

## Caveats

### No Trash

If a file is deleted outside of nextcloud (e.g. when a delete is synced via syncthing, then the file is moved to the `/var/syncthing/versions` directory), it will not be recoverable through the nextcloud interface (it's not in the nextcloud trash).


### Leftover Versions

You'll probably want to disable the [nextcloud version expiration background job](https://docs.nextcloud.com/server/latest/admin_manual/configuration_files/file_versioning.html#background-job), as it won't cleanup versions of files which have been deleted outside of nextcloud.

```
docker exec -it nextcloud occ config:app:set --value=no files_versions background_job_expire_versions
```

Schedule a cronjob to run `/versioner/clean/clean /var/syncthing/versions/var/syncthing/user1` (and also for your other folders) to cleanup the versions. This does cleanup versions of deleted files. However, it will not delete the last version of a deleted file. This is not ideal. You may want to run additional cleanup to free up storage space.

## Tips

- Add `'skeletondirectory' => '',` in `config/www/nextcloud/config/config.php` to disable copying the default 'welcome' files for new user.
- In nextcloud, set quota to `0 B` (for new and existing users), to force user to use external storage only.
- If an External storage folder is shared between users, there's no need to trigger a rescan for each user. It is sufficient to map a single user in the compose file.
- If you have a valid SSL certificate (not self-signed) on your syncthing deployment, then you can remove `- SKIP_SYNCTHING_SSL_VALIDATION=1` from the compose file and redeploy.
- The example compose file and deployment instructions focus on the integration of syncthing with nextcloud external storage. Especially nextcloud may need additional steps to complete setup. Please follow other guides for additional advice and best practices.
- Make `.stfolder` immutable with `chattr +i` to prevent the user from deleting this folder from nextcloud.
- Schedule a cronjob to regularly run `occ files:scan --all` to rescan all nextcloud storage, in case `scanner` missed some syncthing events. For example you could add to `./config/crontabs/root` the following `0	2	*	*	*	/bin/bash -c 'occ files:scan --all && occ preview:pre-generate'` to rescan and pre-generate image previews (if you have the [Preview Generator](https://apps.nextcloud.com/apps/previewgenerator) app installed).