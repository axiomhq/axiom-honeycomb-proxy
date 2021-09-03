# Production image based on alpine.
FROM alpine
LABEL maintainer="Axiom, Inc. <info@axiom.co>"

# Upgrade packages and install ca-certificates.
RUN apk update --no-cache                 \
    && apk upgrade --no-cache             \
    && apk add --no-cache ca-certificates

# Copy binary into image.
COPY axiom-honeycomb-proxy /usr/bin/axiom-honeycomb-proxy

# Use the project name as working directory.
WORKDIR /axiom-honeycomb-proxy

# Expose the default application port.
EXPOSE 8080/tcp

# Set the binary as entrypoint.
ENTRYPOINT [ "/usr/bin/axiom-honeycomb-proxy" ]
