# Background

This is a simple mock service to demonstrate open telemetry instrumentation.

# To compile

```bash
go build main.go server.go -o certinator
```

# To compile for docker

```bash
GOOS=linux go build main.go server.go -o certinator
```

# To run the server

```
HOST_PORT=8080 ./certinator
```

# Useful tip

Docker uses linux subsystem, so if you are running compilation in a mac or pc and wants to run it in docker please use the compile linux command
