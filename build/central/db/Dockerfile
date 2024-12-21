FROM postgres:16-alpine
COPY --chown=postgres:postgres ./postgresql.conf /etc/postgresql.conf
COPY --chown=postgres:postgres schema.sql /docker-entrypoint-initdb.d/schema.sql

CMD [ "-c", "config_file=/etc/postgresql.conf" ]