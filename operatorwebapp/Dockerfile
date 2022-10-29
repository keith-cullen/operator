FROM alpine:latest

# Set the working directory to /app
WORKDIR /app

# Copy the app directory contents into the container at /app
ADD app /app

RUN apk add --no-cache lighttpd

# Make port available to the world outside this container
EXPOSE 8081

CMD lighttpd -D -f lighttpd.conf
