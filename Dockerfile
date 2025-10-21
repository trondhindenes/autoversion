# Final stage - just copy pre-built binary
FROM alpine:latest AS autoversion

# Install git (required for autoversion to work with repositories)
RUN apk add --no-cache git

# Copy the pre-built binary (provided by build context)
COPY autoversion /usr/local/bin/autoversion

# Set working directory
WORKDIR /repo

# Default command
ENTRYPOINT ["/usr/local/bin/autoversion"]

FROM autoversion AS autoversion-action

RUN apk add --no-cache bash
RUN git config --global --add safe.directory '*'

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set working directory
WORKDIR /github/workspace

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]