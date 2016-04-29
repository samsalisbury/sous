# test with docker

Testing service clients means having an instance of the service around.
If you can get a docker container for the service, this'll let you make sure that the container is running,
and get the ip address of the service to then test against.

Given a `docker-compose.yml` file somewhere, you can add something like this to an integration test:

```go
func TestMain(m *testing.M) {
	os.Exit(wrapCompose(m))
}

func wrapCompose(m *testing.M) int {
	ip, started, err := test_with_docker.ComposeServices(
    "docker-machine-name",
    "dir/with/compose-yaml",
    map[string]uint{"ServiceName": 4321}
  )
  // if ComposeServices needs to do a docker-compose up, it will provide an object to use to shut the docker back down
	if started != nil {
		defer test_with_docker.Shutdown(started)
	}

	return m.Run()
}
```

ComposeServices will check to see if any of the named services are missing.
If so, it'll run `docker-compose` in the appropriate directory to bring the services up.
It'll then wait until those services are available before returning,
so you can be sure that your test environment is ready.
If all the named services are already available, ComposeServices returns immediates,
so that you can start things up on your own in order to speed up your test cycle.

Finally, if the services needed to be started for the test, test_with_docker returns a non-nil value
so that you can call Shutdown and clean up afterwards.

For now, test_with_docker assumes Docker Machine and Docker Compose.
Maybe it's worth detecting the need for Machine
and
providing an option for just using `docker up` for simple setups.
