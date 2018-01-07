FROM python:2.7.14-alpine3.7

RUN pip install --upgrade pip
COPY requirements.txt .

RUN echo "ipv6" >> /etc/modules

RUN apk update

RUN apk upgrade --update-cache --available

RUN apk add --update --no-cache g++ gcc libxslt-dev

RUN pip install lxml

RUN apk add --no-cache --update binutils

RUN pip install --no-cache-dir -r requirements.txt
