FROM golang:latest@sha256:4950c1cce16bb754e23ee70d25a67e906149d0faedc0aaeed49c075b696fa889

WORKDIR /app

# Set Go environment variables
RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go env -w CGO_ENABLED=1

# Copy everything from the current directory to the working directory in the container
COPY . .

# Change permissions for the script
RUN chmod +x /app/src/raft/loop_test.sh

# Set the entrypoint to execute the script

ENTRYPOINT ["sh", "-c", "cd /app/src/raft && bash loop_test.sh"]

