FROM ubuntu

COPY ./ekpose /usr/local/bin/ekpose

CMD  [ "/usr/local/bin/ekpose" ]