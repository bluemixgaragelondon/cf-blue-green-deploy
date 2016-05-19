FROM golang:1.5.4

RUN wget "https://cli.run.pivotal.io/stable?version=6.18.0&release=linux64-binary" -O - | tar zxvf -
RUN chmod 755 cf
RUN mv cf /usr/local/bin
