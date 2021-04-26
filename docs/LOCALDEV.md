# How to start documentation container locally?

Don't forget to clone repo firstly.

1. Free 80 port to bind

Stop other service or make appropriate changes to the commands below.

2. Open separate console and start container
```shell
cd docs
source $(multiwerf use 1.2 alpha --as-file)
werf run --follow --docker-options="-d --name trdl"
```

> Or run the container on the other port, for instanse `werf run --follow --docker-options="-p 82:80 -d --name trdl"

