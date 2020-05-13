# Troubleshooting

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


