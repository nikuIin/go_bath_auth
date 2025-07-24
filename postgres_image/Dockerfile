FROM postgres:17.4-bookworm

# For right local working
RUN apt-get update && \
    apt-get install -y locales && \
    rm -rf /var/lib/apt/lists/*

# Generic ru_RU loccal
RUN localedef -i ru_RU -c -f UTF-8 -A /usr/share/locale/locale.alias ru_RU.UTF-8

# Set locals to env
ENV LANG ru_RU.utf8
