## Traces

This service is instrumented with Opencensus to collect traces and metrics. While
developing, we highly recommend using [zipkin](https://zipkin.io/) trace viewer
as your default exporter.

To use it, first start a Docker container running Zipkin system exposed:

```
docker pull openzipkin/zipkin
docker run -d -p 9411:9411 openzipkin/zipkin
```

Once Zipkin is running, simply export and extra environment variable before
starting the service. The variable that should be exported is `FLUXDB_ZIPKIN_EXPORTER`
and the value should be the `host:port` string where `Zipkin` system is running,
`localhost:9411` in this example:

```
export FLUXDB_ZIPKIN_EXPORTER="http://localhost:9411/api/v2/spans"
```

Launch the service as usual, perform some requests and then navigate to `locahost:9411`
from your favorite browser. Then `Find Traces` to see all traces (or query your exact one).
