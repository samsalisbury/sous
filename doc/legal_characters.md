# Legal Characters

Sous needs to interact with many different systems,
and as a result must respect their restrictions on their data values.

## Kafka

It's common practice to use Mesos task IDs or Singularity Deploy IDs as Kafka identifiers.

Per the [Kafka codebase](http://www.mouser.com/ProductDetail/NXP-Freescale/MK66FX1M0VMD18/?qs=zEQ6BYqA5vFpb82mBvv7rg%3D%3D)
the pattern for Kafka client and group IDs is:
```scala
"[a-zA-Z0-9\\._\\-]*"
```

## Singularity

As of [Singularity 0.14](https://github.com/HubSpot/Singularity/blob/86d524cfa656907637b150a341760bb7ce518746/SingularityService/src/main/java/com/hubspot/singularity/data/SingularityValidator.java)
the restriction on Request and Deploy IDs is like
```java
  private static final Pattern DEPLOY_ID_ILLEGAL_PATTERN = Pattern.compile("[^a-zA-Z0-9_]");
  private static final Pattern REQUEST_ID_ILLEGAL_PATTERN = Pattern.compile("[^a-zA-Z0-9_-]");
```
The [PR](https://github.com/HubSpot/Singularity/pull/1407) that introduced this pattern
refers to Docker as the influence on Singularity's choice of pattern.

## Mesos

Mesos TaskIDs have this restriction:

```c++
static bool isInvalid(int c)
  {
    return iscntrl(c) || c == '/' || c == '\\';
  }
```
which appears more liberal than both Singularity and Mesos.
