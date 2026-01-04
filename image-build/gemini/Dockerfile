# syntax=docker/dockerfile:1
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG GEMINI_CLI_VERSION=latest
ARG SANDBOX_NAME="gemini-cli-sandbox"
ENV SANDBOX="$SANDBOX_NAME"

# install gemini-cli and clean up
RUN npm install -g @google/gemini-cli@${GEMINI_CLI_VERSION} \
  && npm cache clean --force

# default entrypoint when none specified
CMD ["gemini"]
