FROM mysql/mysql-server
MAINTAINER Roman Gelembjuk

# Create MySQL DB
ENV MYSQL_ROOT_PASSWORD rootpassword
ENV MYSQL_DATABASE BC
ENV MYSQL_USER blockchain
ENV MYSQL_PASSWORD blockchain

ADD node /
ADD docker/conf /conf
EXPOSE 8765
EXPOSE 8766

ENTRYPOINT ["/node"]
CMD ["interactiveautocreate"]
