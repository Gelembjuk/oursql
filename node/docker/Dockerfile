FROM ubuntu:latest
MAINTAINER Roman Gelembjuk

# Install latest updates
RUN apt-get update
RUN apt-get upgrade -y

# Install mysql client and server
RUN DEBIAN_FRONTEND=noninteractive apt-get -y install mysql-client mysql-server iproute2

# Create temp file to know about the first start of a container
RUN touch firststart.lock

# Set Standard settings
ENV DBNAME BC
ENV DBUSER blockchain
ENV DBPASSWORD blockchain

ADD node /
ADD docker/conf/ /conf

ADD docker/run.sh /usr/local/bin/run.sh
RUN chmod +x /usr/local/bin/run.sh

ADD docker/healthcheck.sh /usr/local/bin/healthcheck.sh
RUN chmod +x /usr/local/bin/healthcheck.sh

HEALTHCHECK CMD /usr/local/bin/healthcheck.sh

EXPOSE 8765
EXPOSE 8766

ENTRYPOINT ["/usr/local/bin/run.sh"]
CMD ["interactiveautocreate"]
