FROM ubuntu:18.04
# install curl, netstat, ping, vim
RUN apt-get update && apt-get install -y \
    ca-certificates curl net-tools iputils-ping vim \
    && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/sqleton"]
EXPOSE 8080
COPY sqleton /sqleton