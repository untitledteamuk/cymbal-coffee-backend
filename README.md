# Cymbal Coffee Backend Beans

Welcome Cymbal Coffee Competitor! This is the app you will be using to win our business!

No, it's not written in Java (although that would have been on theme), but it's a simple Go app. You don't need to know that though, you just need to get it running!

## How do I get started?

It's up to you! This is a basic Go app and we've included a `Dockerfile` in case you wish to containerise it. We've also included a simple `app.yaml` in case you want to run it somewhere else.

The app has virtually no requirements at first. It's stateless and can be run many times. It will write various logs about what's happening, so if things aren't working maybe check them out.

If you're really stuck, please ask one of your specialists for help!

## How does the app work?

To score our competitors we have a spy service - _bond_ - that this app communicates with. If it can't reach bond - or if it's not been registered - the app will crash. Your tasks that you run through on the day will validate their configuration with Bond.

You can take a look at its source code if you like. Maybe there are bonus points for hacking it? Maybe you'll lose points instead!

## Testing

To test locally, please ensure that you have the following dependencies installed:

1. [Docker](https://docs.docker.com/engine/install)
2. [Go](https://go.dev/)

Alternatively, Docker (and go) comes pre-installed on Google Cloud Workstations, and in Google Cloud Shell. Feel free to run the tests using Google Cloud.

### Testing with Go

To run our tests using go:

```bash
go test -v
```

### Testing with Docker

To run our tests in Docker, there is an included file apt-ly named [Dockerfile.qa](Dockerfile.qa). This can be built and run locally or used in a CI/CD pipeline.

When running in Google Cloud Shell the project id is a local ENV named `GOOGLE_CLOUD_PROJECT`.

```bash
docker build -t backend-service-test:latest --build-arg project_id="$GOOGLE_CLOUD_PROJECT" -f Dockerfile.qa .
```

Run the container:

```bash
docker run backend-service-test:latest
```

### Testing with Scripts

To run our tests using the included script:

```bash
./test.sh
```
