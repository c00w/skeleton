# Docker build file for creating postgresql-9.2 container
FROM base


# instructions used from:
#   http://docs.docker.io/en/latest/examples/postgresql_service/
# alternatively, maybe just do sudo docker pull zaiste/postgresql


RUN sudo docker run -i -t ubuntu /bin/bash
# ^may need to add -dns=<local DNS>

RUN apt-get update
RUN apt-get -y install python-software-properties
RUN apt-get -y install software-properties-common

# see http://wiki.postgresql.org/wiki/Apt for info on postgresql repo
#   might need to change this command based on ubuntu version
RUN add-apt-repository ppa:pitti/postgresql
# ^need to press ENTER here to continue (warning about deprecation of #   ppa)
RUN apt-get update 

RUN apt-get -y install postgresql-9.2 postgresql-client-9.2 postgresql-contrib-9.2

# Create PostgreSQL superuser role
RUN su postgres -c "createuser -P -d -r -s docker"
# ^user must input docker as password
# Create test database 
RUN su postgres -c "createdb -O docker docker"


RUN sed 's/127.0.0.1_/32/0.0.0.0_/0' /etc/postgresql/9.2/main/pg_hba.conf

RUN sed 's/#listen/listen' /etc/postgresql/9.2/main/postgresql.conf
RUN sed 's/localhost/*' /etc/postgresql/9.2/main/postgresql.conf

# ^look at PostgreSQL documentation to fine-tune for security issues


RUN exit


# create image
docker commit <container_id> username/postgresql

# run postgresql server
RUN CONTAINER=$(sudo docker run -d -p 5432 \
  -t username/postgresql \
  /bin/su postgres -c '/usr/lib/postgresql/9.2/bin/postgres \
    -D /var/lib/postgresql/9.2/main \
    -c config_file=/etc/postgresql/9.2/main/postgresql.conf')
	
# connect to server using psql	
RUN	CONTAINER_IP=$(sudo docker inspect $CONTAINER | grep IPAddress | awk '{ print $2 }' | tr -d ',"')

psql -h $CONTAINER_IP -p 5432 -d docker -U docker -W
