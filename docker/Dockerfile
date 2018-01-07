FROM python:2.7.14-alpine3.7

RUN pip install --upgrade pip
COPY requirements.txt .

RUN echo "ipv6" >> /etc/modules

RUN apk update

RUN pip install --no-cache-dir -r requirements.txt
