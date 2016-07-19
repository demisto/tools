FROM python:2
MAINTAINER Slavik Markovich <s@demisto.com>

# Install PhantomJS, rasterize.js

RUN mkdir /tmp/phantomjs \
    && curl -L https://bitbucket.org/ariya/phantomjs/downloads/phantomjs-2.1.1-linux-x86_64.tar.bz2 | tar -xj --strip-components=1 -C /tmp/phantomjs \
    && mv /tmp/phantomjs/bin/phantomjs /usr/local/bin \
    && curl https://raw.githubusercontent.com/ariya/phantomjs/master/examples/rasterize.js > /usr/local/bin/rasterize.js

CMD ["python2"]