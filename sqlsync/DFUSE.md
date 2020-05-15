

Port forwards:

```bash

kc port-forward svc/blockmeta-v2 9000 & kc port-forward svc/relayer-v2 9001:9000 & kc port-forward svc/fluxdb-server-v2 9002:80 &

```
