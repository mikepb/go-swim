# Algorithms


# Round-trip time estimation

From Section 3.5 of _Computer Networking: A Top-Down Approach_ by Kurose and Ross.

```go
EstimatedRTT = time.Duration(
    (1.0 - Alpha) * float64(EstimatedRTT) + Alpha * float64(SampleRTT))

DeviationRTT = time.Duration(
    (1.0 - Beta) * float64(DeviationRTT) + 
    Beta * math.Abs(float64(SampleRTT - EstimatedRTT))
```

RFC 6928 recommends `Alpha = .125` and `Beta = .25`
