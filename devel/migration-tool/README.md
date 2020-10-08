# Migration Tool Demo

1) Create the migration data from a state snapshot
```shell script
./start.sh -m export
```

2) Edit the migration data
```shell script
cd data-editor
yarn run start
```

3) Migrate the edited data to a new chain
```shell script
./start.sh -m import
```

