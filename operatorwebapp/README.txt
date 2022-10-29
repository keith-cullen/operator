$ docker build -t localhost:5000/operatorwebapp:latest .
$ docker run -d -p 8081:8081 localhost:5000/operatorwebapp:latest
$ docker container stop <CONTAINER-ID>
$ docker container rm <CONTAINER-ID>
$ docker image rm <IMAGE-ID>
$ docker push localhost:5000/operatorwebapp:latest
