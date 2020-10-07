# TST
Tst is a tool for performing incremental backups under hood using [rclone](https://rclone.org/) utility.

## Building
You must have Golang 1.15 compiler at least to build tst.
Then run following:
```
cd cmd/tst && go build .
```
You will build executable named `tst`. Be sure to put it in the PATH.

## Usage
First of all, you need to setup using `rclone config`. For more information about configuration, please refer to the [rclone docs](https://rclone.org/).

### Help
```
Backup utility under the hood using rclone.

Usage:
   [flags]
   [command]

Available Commands:
  help          Help about any command
  init          Initialize an empty collection at current directory.
  list-versions List all versions for current collection.
  restore       Restores specified version.
  version       Creates a new version of current collection and pushes it.

Flags:
  -h, --help   help for this command
```

### Collections
When you use tst, you manipulate *collections* — a set of files (like git repository).
To initialize collection, create a directory and then execute `tst init` there:
```
mkdir collection
tst init remote:remote-test/
```
Syntax for second argument is `<rclone backend name:remote root>`.

### Versions
Version is a snapshot of current collections. To make version just run `tst version <comment>`. This will snapshot all files in current collection recursively and push all new data to the remote storage. You could use `comment` to give some information about new version (like git commits).

To restore collection to the specified version, run `tst restore v0` — this will restore initial version of files.
All versions are numbered from 0 to infinity. To see all versions and their messages run `tst list-versions`.
