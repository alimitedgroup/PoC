FROM postgres:16-alpine
COPY --chown=postgres:postgres ./postgresql.conf /etc/postgresql.conf
CMD [ "-c", "config_file=/etc/postgresql.conf" ]