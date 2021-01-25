# Adjust timestamp processor

Takes `sumologic.telemetry.sdk.export_timestamp` attribute and compares it against
actually recorded timestamp of package retrieval to correct span start/end
times.

## Configuration

The `threshold` property (default: 10s) defines the minimum difference required to adjust the span timestamp

```yaml
processors:
  adjustts:
    threshold: 10s
```
