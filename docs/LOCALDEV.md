# How to start documentation container locally?

Don't forget to clone repo firstly.

1. Free 80 port to bind

Stop other service or make appropriate changes to the commands below.

2. Open separate console and start container
```shell
cd docs
source $(dl use werf 1.2 ea)
werf run --follow --docker-options="-d --name trdl -p 80:80" --dev web
```

