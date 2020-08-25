# Troubleshooting

## installing / compiling error

If you're getting one or more error message like:
* `warning Error running install script for optional dependency:`
* `No receipt for 'com.apple.pkg.CLTools_Executables' found at '/'.`
* `gyp: No Xcode or CLT version detected!`

It probably means that your local Command Line Tools for Xcode was somehow corrupted. You should do a clean uninstall of Command Line Tools for Xcode, and then reinstall Command Line Tools for Xcode. This article should point you in the right direction: [How to resolve, No Xcode or CLT version detected!](https://medium.com/@mrjohnkilonzi/how-to-resolve-no-xcode-or-clt-version-detected-d0cf2b10a750).

## failed continuity check (mindreader)

* **Symptom**:
  * Mindreader instance refuses to start
* **Log messages**: 
  * `{"error": "continuityChecker failed: block 1911 would creates a hole after highest seen block: 1909"}`
  * `{"error": "continuityChecker already locked"}`
* **Cause**: The mindreader process missed a few blocks (probably because of an unclean shutdown) and the nodeos instance is passed that "hole". A manual restore operation is needed.
* **Solution**: Call the 'snapshot_restore' endpoint on the Mindreader manager to initiate a restore from latest snapshot, while dfuse-eos is running:

```
curl -sS -XPOST localhost:13009/v1/snapshot_restore
```


